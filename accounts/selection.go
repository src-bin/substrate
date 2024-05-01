package accounts

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

type AccountWithSelectors struct {
	Account   *awsorgs.Account
	Selectors []string
}

type Selection struct {
	AllDomains bool
	Domains    []string

	AllEnvironments bool
	Environments    []string

	AllQualities bool
	Qualities    []string

	Substrate bool
	Humans    bool `json:"-"` // not exposed in arguments; like Substrate but without arbitrary policy attachments

	Management bool

	Specials []string

	Numbers []string // raw 12-digit AWS account numbers
}

func (s *Selection) Arguments() []string {
	var ss []string

	if s.AllDomains {
		ss = append(ss, "--all-domains")
	} else {
		for _, domain := range s.Domains {
			ss = append(ss, "--domain", fmt.Sprintf("%q", domain))
		}
	}

	if s.AllEnvironments {
		ss = append(ss, "--all-environments")
	} else {
		for _, environment := range s.Environments {
			ss = append(ss, "--environment", fmt.Sprintf("%q", environment))
		}
	}

	if s.AllQualities {
		ss = append(ss, "--all-qualities")
	} else {
		for _, quality := range s.Qualities {
			ss = append(ss, "--quality", fmt.Sprintf("%q", quality))
		}
	}

	if s.Substrate {
		ss = append(ss, "--substrate")
	}

	// Don't translate s.Humans into --humans because that argument is handled
	// in ManagedAssumeRolePolicy. In Selection it's purely behind the scenes.
	/*
		if s.Humans {
			ss = append(ss, "--humans")
		}
	*/

	if s.Management {
		ss = append(ss, "--management")
	}

	for _, special := range s.Specials {
		ss = append(ss, "--special", fmt.Sprintf("%q", special))
	}

	for _, number := range s.Numbers {
		ss = append(ss, "--number", fmt.Sprintf("%q", number))
	}

	return ss
}

func (s *Selection) FlagSet(u SelectionFlagsUsage) *pflag.FlagSet {
	if u.AllDomains == "" {
		panic("SelectionFlagsUsage.AllDomains can't be empty")
	}
	if u.Domains == "" {
		panic("SelectionFlagsUsage.Domains can't be empty")
	}
	if u.AllEnvironments == "" {
		panic("SelectionFlagsUsage.AllEnvironments can't be empty")
	}
	if u.Environments == "" {
		panic("SelectionFlagsUsage.Environments can't be empty")
	}
	if u.AllQualities == "" {
		panic("SelectionFlagsUsage.AllQualities can't be empty")
	}
	if u.Qualities == "" {
		panic("SelectionFlagsUsage.Qualities can't be empty")
	}
	if u.Substrate == "" {
		panic("SelectionFlagsUsage.Substrate can't be empty")
	}
	if u.Management == "" {
		panic("SelectionFlagsUsage.Management can't be empty")
	}
	if u.Specials == "" {
		panic("SelectionFlagsUsage.Specials can't be empty")
	}
	if u.Numbers == "" {
		panic("SelectionFlagsUsage.Numbers can't be empty")
	}
	s.Reset()
	set := pflag.NewFlagSet("[account selection flags]", pflag.ExitOnError)
	set.BoolVar(&s.AllDomains, "all-domains", false, u.AllDomains)
	set.StringArrayVarP(&s.Domains, "domain", "d", []string{}, u.Domains)
	set.BoolVar(&s.AllEnvironments, "all-environments", false, u.AllEnvironments)
	set.StringArrayVarP(&s.Environments, "environment", "e", []string{}, u.Environments)
	set.BoolVar(&s.AllQualities, "all-qualities", false, u.AllQualities)
	set.StringArrayVar(&s.Qualities, "quality", []string{}, u.Qualities)
	set.BoolVar(&s.Substrate, "substrate", false, u.Substrate)
	set.BoolVar(&s.Management, "management", false, u.Management)
	set.StringArrayVar(&s.Specials, "special", []string{}, u.Specials)
	set.StringArrayVar(&s.Numbers, "number", []string{}, u.Numbers)
	return set
}

func (s *Selection) Match(a *awsorgs.Account) (selectors []string, ok bool) {
	ok = true

	if s.AllDomains {
		selectors = append(selectors, "all-domains")
	} else if len(s.Domains) > 0 {
		if contains(s.Domains, a.Tags[tagging.Domain]) {
			selectors = append(selectors, "domain")
		} else {
			ok = false
		}
	} else {
		ok = false
	}

	if s.AllEnvironments {
		selectors = append(selectors, "all-environments")
	} else if len(s.Environments) > 0 {
		if contains(s.Environments, a.Tags[tagging.Environment]) {
			selectors = append(selectors, "environment")
		} else {
			ok = false
		}
	} else {
		ok = false
	}

	if s.AllQualities {
		selectors = append(selectors, "all-qualities")
	} else if len(s.Qualities) > 0 {
		if contains(s.Qualities, a.Tags[tagging.Quality]) {
			selectors = append(selectors, "quality")
		} else {
			ok = false
		}
	} else {
		ok = false
	}

	return
}

func (s *Selection) Partition(ctx context.Context, cfg *awscfg.Config) (
	selected []AccountWithSelectors,
	unselected []*awsorgs.Account,
	err error,
) {
	// TODO there's some redundancy in Grouped and Partition which maybe can be rectified later
	adminAccounts, serviceAccounts, substrateAccount, auditAccount, deployAccount, managementAccount, networkAccount, err := Grouped(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	for _, account := range serviceAccounts {
		if account.Tags[tagging.Domain] == "" {
			continue // don't overreach into not-quite-Substrate-managed accounts
		}
		if selectors, ok := s.Match(account); ok {
			selected = append(selected, AccountWithSelectors{
				Account:   account,
				Selectors: selectors,
			})
		} else {
			unselected = append(unselected, account)
		}
	}

	if s.Substrate || s.Humans {
		var selectors []string
		if s.Substrate && s.Humans {
			selectors = []string{"substrate", "humans"}
		} else if s.Substrate {
			selectors = []string{"substrate"}
		} else if s.Humans {
			selectors = []string{"humans"}
		}
		if substrateAccount != nil {
			selected = append(selected, AccountWithSelectors{
				Account:   substrateAccount,
				Selectors: selectors,
			})
		}
		for _, account := range adminAccounts {
			selected = append(selected, AccountWithSelectors{
				Account:   account,
				Selectors: selectors,
			})
		}
	} else {
		if substrateAccount != nil {
			unselected = append(unselected, substrateAccount)
		}
		unselected = append(unselected, adminAccounts...)
	}

	if s.Management {
		selected = append(selected, AccountWithSelectors{
			Account:   managementAccount,
			Selectors: []string{"management"},
		})
	} else {
		unselected = append(unselected, managementAccount)
	}

	var selectedAudit, selectedDeploy, selectedNetwork bool
	for _, special := range s.Specials {
		switch special {
		case Audit:
			selected = append(selected, AccountWithSelectors{
				Account:   auditAccount,
				Selectors: []string{"special"},
			})
			selectedAudit = true
		case Deploy:
			selected = append(selected, AccountWithSelectors{
				Account:   deployAccount,
				Selectors: []string{"special"},
			})
			selectedDeploy = true
		case Network:
			selected = append(selected, AccountWithSelectors{
				Account:   networkAccount,
				Selectors: []string{"special"},
			})
			selectedNetwork = true
		default:
			return nil, nil, SelectionError("creating additional roles in the audit account is not supported")
		}
	}
	if !selectedAudit {
		unselected = append(unselected, auditAccount)
	}
	if !selectedDeploy {
		unselected = append(unselected, deployAccount)
	}
	if !selectedNetwork {
		unselected = append(unselected, networkAccount)
	}

	if len(s.Numbers) > 0 {
		ui.Print("warning: `substrate role list` and `substrate role delete` will not be able to find roles created in numbered accounts; if you wish to delete them in the future you will have to do so yourself")
		for _, number := range s.Numbers {
			selected = append(selected, AccountWithSelectors{
				Account:   awsorgs.StringableZeroAccount(number),
				Selectors: []string{"number"},
			})
		}
	}

	return
}

func (s *Selection) Reset() {
	s.AllDomains = false
	s.Domains = []string{}
	s.AllEnvironments = false
	s.Environments = []string{}
	s.AllQualities = false
	s.Qualities = []string{}
	s.Substrate = false
	s.Humans = false
	s.Management = false
	s.Specials = []string{}
	s.Numbers = []string{}
}

func (s *Selection) Sort() error {
	sort.Strings(s.Domains)
	environments, err := naming.Environments()
	if err != nil {
		return err
	}
	naming.IndexedSort(s.Environments, environments)
	qualities, err := naming.Qualities()
	if err != nil {
		return err
	}
	naming.IndexedSort(s.Qualities, qualities)
	sort.Strings(s.Specials)
	sort.Strings(s.Numbers)
	return nil
}

func (s *Selection) String() string {
	return strings.Join(s.Arguments(), " ")
}

func (s *Selection) Validate() error {
	if !s.AllDomains &&
		len(s.Domains) == 0 &&
		!s.AllEnvironments &&
		len(s.Environments) == 0 &&
		!s.AllQualities &&
		len(s.Qualities) == 0 &&
		!s.Substrate &&
		!s.Humans &&
		!s.Management &&
		len(s.Specials) == 0 &&
		len(s.Numbers) == 0 {
		return SelectionError("at least one account selection flag is required")
	}

	// If no explicit --quality was given and we only have one quality,
	// imply --all-qualities.
	// TODO also imply --all-qualities if there is only one valid quality
	// for each valid environment.
	if s.Qualities == nil || len(s.Qualities) == 0 {
		qualities, err := naming.Qualities()
		if err != nil {
			return err
		}
		if len(qualities) == 1 {
			s.AllQualities = true
		}
	}

	return nil
}

type SelectionError string

func (err SelectionError) Error() string {
	return fmt.Sprint("SelectionError: ", string(err))
}

type SelectionFlagsUsage struct {
	AllDomains string
	Domains    string

	AllEnvironments string
	Environments    string

	AllQualities string
	Qualities    string

	Substrate string

	Management string

	Specials string

	Numbers string
}

func contains(ss []string, s string) bool {
	for i := 0; i < len(ss); i++ {
		if ss[i] == s {
			return true
		}
	}
	return false
}
