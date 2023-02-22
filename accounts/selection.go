package accounts

import (
	"context"
	"flag"
	"fmt"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

type AccountWithSelectors struct {
	Account   *awsorgs.Account
	Selectors []string
}

type Selection struct {
	Admin bool

	AllDomains bool
	Domains    []string

	AllEnvironments bool
	Environments    []string

	AllQualities bool
	Qualities    []string

	Management bool

	Specials []string

	Numbers []string // raw 12-digit AWS account numbers
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
	adminAccounts, serviceAccounts, _, deployAccount, managementAccount, networkAccount, err := Grouped(ctx, cfg)
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

	if s.Management {
		selected = append(selected, AccountWithSelectors{
			Account:   managementAccount,
			Selectors: []string{"management"},
		})
	} else {
		unselected = append(unselected, managementAccount)
	}

	var selectedDeploy, selectedNetwork bool
	for _, special := range s.Specials {
		switch special {
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
	if !selectedDeploy {
		unselected = append(unselected, deployAccount)
	}
	if !selectedNetwork {
		unselected = append(unselected, networkAccount)
	}

	if len(s.Numbers) > 0 {
		ui.Print("warning: `substrate roles` and `substrate delete-role` will not be able to find roles created in numbered accounts; if you wish to delete them in the future you will have to do so yourself")
		for _, number := range s.Numbers {
			selected = append(selected, AccountWithSelectors{
				Account:   awsorgs.StringableZeroAccount(number),
				Selectors: []string{"number"},
			})
		}
	}

	return
}

type SelectionError string

func (err SelectionError) Error() string {
	return fmt.Sprint("SelectionError: ", string(err))
}

type Selector struct {
	AllDomains *bool
	Domains    *cmdutil.StringSliceFlag

	AllEnvironments *bool
	Environments    *cmdutil.StringSliceFlag

	AllQualities *bool
	Qualities    *cmdutil.StringSliceFlag

	Admin *bool

	Management *bool

	Specials *cmdutil.StringSliceFlag

	Numbers *cmdutil.StringSliceFlag
}

func NewSelector(su SelectorUsage) *Selector {
	return &Selector{
		AllDomains: flag.Bool("all-domains", false, su.AllDomains),
		Domains:    cmdutil.StringSlice("domain", su.Domains),

		AllEnvironments: flag.Bool("all-environments", false, su.AllEnvironments),
		Environments:    cmdutil.StringSlice("environment", su.Environments),

		AllQualities: flag.Bool("all-qualities", false, su.AllQualities),
		Qualities:    cmdutil.StringSlice("quality", su.Qualities),

		Admin: flag.Bool("admin", false, su.Admin),

		Management: flag.Bool("management", false, su.Management),

		Specials: cmdutil.StringSlice("special", su.Specials),

		Numbers: cmdutil.StringSlice("number", su.Numbers),
	}
}

func (s *Selector) Selection() (*Selection, error) {
	if !flag.Parsed() {
		panic("(*Selector).Selection called before flag.Parse")
	}

	// TODO validation and maybe return nil, SelectionError("...")

	// If no explicit -quality was given and we only have one quality,
	// imply -all-qualities.
	if s.Qualities.Len() == 0 {
		if qualities, _ := naming.Qualities(); len(qualities) == 1 {
			*s.AllQualities = true
		}
	}

	return &Selection{
		AllDomains: *s.AllDomains,
		Domains:    s.Domains.Slice(), // TODO expand if AllDomains == true?

		AllEnvironments: *s.AllEnvironments,
		Environments:    s.Environments.Slice(), // TODO expand if AllEnvironments == true?

		AllQualities: *s.AllQualities,
		Qualities:    s.Qualities.Slice(), // TODO expand if AllQualities == true?
		// TODO do we need to do anything to cover the special case of only having one quality which may be omitted in single-account selection contexts?

		Admin: *s.Admin,

		Management: *s.Management,

		Specials: s.Specials.Slice(),

		Numbers: s.Numbers.Slice(),
	}, nil
}

type SelectorUsage struct {
	Admin string

	AllDomains string
	Domains    string

	AllEnvironments string
	Environments    string

	AllQualities string
	Qualities    string

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
