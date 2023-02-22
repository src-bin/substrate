package accounts

import (
	"flag"
	"fmt"

	"github.com/src-bin/substrate/cmdutil"
)

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
