package terraform

import (
	"fmt"

	"github.com/src-bin/substrate/version"
)

// EC2Tag generates the aws_ec2_tag resource, useful for tagging VPCs created
// in another account and shared into this one (because their tags don't get
// shared along).
type EC2Tag struct {
	ForEach    Value
	Key, Value Value
	Label      Value
	Provider   ProviderAlias
	ResourceId Value
}

func (t EC2Tag) Ref() Value {
	return Uf("aws_ec2_tag.%s", t.Label)
}

func (EC2Tag) Template() string {
	return `resource "aws_ec2_tag" {{.Label.Value}} {
{{- if .ForEach}}
  for_each = {{.ForEach.Value}}
{{- end}}
  key = {{.Key.Value}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  resource_id = {{.ResourceId.Value}}
  value = {{.Value.Value}}
}
`
}

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
