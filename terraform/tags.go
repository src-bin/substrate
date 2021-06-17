package terraform

import (
	"fmt"

	"github.com/src-bin/substrate/version"
)

type Tags struct {
	Connectivity                 string // "public" or "private"; used only by subnets
	Domain, Environment, Quality string
	Name                         string
	Region, AvailabilityZone     string
	SkipMeta                     bool // used by data sources querying by tags
	Special                      string
}

func (Tags) Manager() string { return "Terraform" }

func (Tags) SubstrateVersion() string { return version.Version }

func (t Tags) Value() Value {
	format := "\n    %s = %q"
	s := "  {"
	if t.AvailabilityZone != "" {
		s += fmt.Sprintf(format, "AvailabilityZone", t.AvailabilityZone)
	}
	if t.Connectivity != "" {
		s += fmt.Sprintf(format, "Connectivity", t.Connectivity)
	}
	if t.Domain != "" {
		s += fmt.Sprintf(format, "Domain", t.Domain)
	}
	if t.Environment != "" {
		s += fmt.Sprintf(format, "Environment", t.Environment)
	}
	if !t.SkipMeta {
		s += fmt.Sprintf(format, "Manager", t.Manager())
	}
	if t.Name != "" {
		s += fmt.Sprintf(format, "Name", t.Name)
	}
	if t.Quality != "" {
		s += fmt.Sprintf(format, "Quality", t.Quality)
	}
	if t.Region != "" {
		//s += fmt.Sprintf(format, "Region", t.Region)
	}
	if t.Special != "" {
		//s += fmt.Sprintf(format, "Special", t.Special)
	}
	if !t.SkipMeta {
		s += fmt.Sprintf(format, "SubstrateVersion", t.SubstrateVersion())
	}
	s += "\n  }"
	return U(s)
}
