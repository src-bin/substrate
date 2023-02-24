package accounts

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

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
	AllDomains bool
	Domains    []string

	AllEnvironments bool
	Environments    []string

	AllQualities bool
	Qualities    []string

	Admin bool

	Management bool

	Specials []string

	Numbers []string // raw 12-digit AWS account numbers
}

func (s *Selection) Arguments() []string {
	var ss []string

	if s.AllDomains {
		ss = append(ss, "-all-domains")
	} else {
		for _, domain := range s.Domains {
			ss = append(ss, "-domain", domain)
		}
	}

	if s.AllEnvironments {
		ss = append(ss, "-all-environments")
	} else {
		for _, environment := range s.Environments {
			ss = append(ss, "-environment", environment)
		}
	}

	if s.AllQualities {
		ss = append(ss, "-all-qualities")
	} else {
		for _, quality := range s.Qualities {
			ss = append(ss, "-quality", quality)
		}
	}

	if s.Admin {
		ss = append(ss, "-admin")
	}

	if s.Management {
		ss = append(ss, "-management")
	}

	for _, special := range s.Specials {
		ss = append(ss, "-special", special)
	}

	for _, number := range s.Numbers {
		ss = append(ss, "-number", number)
	}

	return ss
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

	// Basically no one is going to write -domain "admin" -environment "admin"
	// to select admin accounts. They're going to write -admin. And we should
	// be smart enough to parrot that back to them in `substrate roles`, even
	// if it's syntactic sugar.
	if a.Tags[tagging.Domain] == naming.Admin && a.Tags[tagging.Environment] == naming.Admin && len(selectors) == 2 && selectors[0] == "domain" && selectors[1] == "environment" {
		selectors = []string{"admin"}
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

	for _, account := range adminAccounts {
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

type SelectionError string

func (err SelectionError) Error() string {
	return fmt.Sprint("SelectionError: ", string(err))
}

type SelectionFlags struct {
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

func NewSelectionFlags(u SelectionFlagsUsage) *SelectionFlags {
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
	if u.Admin == "" {
		panic("SelectionFlagsUsage.Admin can't be empty")
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
	return &SelectionFlags{
		AllDomains: flag.Bool("all-domains", false, u.AllDomains),
		Domains:    cmdutil.StringSlice("domain", u.Domains),

		AllEnvironments: flag.Bool("all-environments", false, u.AllEnvironments),
		Environments:    cmdutil.StringSlice("environment", u.Environments),

		AllQualities: flag.Bool("all-qualities", false, u.AllQualities),
		Qualities:    cmdutil.StringSlice("quality", u.Qualities),

		Admin: flag.Bool("admin", false, u.Admin),

		Management: flag.Bool("management", false, u.Management),

		Specials: cmdutil.StringSlice("special", u.Specials),

		Numbers: cmdutil.StringSlice("number", u.Numbers),
	}
}

func (f *SelectionFlags) Selection() (*Selection, error) {
	if !flag.Parsed() {
		panic("(*SelectionFlags).Selection called before flag.Parse")
	}

	// If no explicit -quality was given and we only have one quality,
	// imply -all-qualities.
	if f.Qualities.Len() == 0 {
		if qualities, _ := naming.Qualities(); len(qualities) == 1 {
			*f.AllQualities = true
		}
	}

	// If -admin was given and either of -all-domains or -all-environments
	// weren't, add "admin" to the selected domains and/or environments so
	// that the matcher will select admin accounts, too.
	if *f.Admin {
		if !*f.AllDomains {
			f.Domains.Set(naming.Admin)
		}
		if !*f.AllEnvironments {
			f.Environments.Set(naming.Admin)
		}
	}

	// TODO validation and maybe return nil, SelectionError("...")

	return &Selection{
		AllDomains: *f.AllDomains,
		Domains:    f.Domains.Slice(),

		AllEnvironments: *f.AllEnvironments,
		Environments:    f.Environments.Slice(),

		AllQualities: *f.AllQualities,
		Qualities:    f.Qualities.Slice(),

		Admin: *f.Admin,

		Management: *f.Management,

		Specials: f.Specials.Slice(),

		Numbers: f.Numbers.Slice(),
	}, nil
}

type SelectionFlagsUsage struct {
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
