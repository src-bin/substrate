package terraform

import (
	"fmt"

	"github.com/src-bin/substrate/version"
)

type Provider struct {
	Alias, AliasPrefix, AliasSuffix, Region string // if unset, Alias is constructed from the other three
	RoleArn                                 string
	SessionName, ExternalId                 string
}

// NetworkProviderFor returns a Terraform provider for discovering the VPCs and
// subnets in the given region's network.  The given sess must be in the management
// account (in any role).
func NetworkProviderFor(region, roleArn string) Provider {
	return Provider{
		Alias:       "network",
		Region:      region,
		RoleArn:     roleArn,
		SessionName: "Terraform",
	}
}

// ProviderFor returns a Terraform provider that assumes the Administrator role
// in sess's account in the given region.
func ProviderFor(region, roleArn string) Provider {
	return Provider{
		Region:      region,
		RoleArn:     roleArn,
		SessionName: "Terraform",
	}
}

// UsEast1Provider returns a Terraform provider that assumes the Administrator
// role in sess's account in us-east-1 where it can configure services that are
// exclusively offered in us-east-1 such as ACM certificates for CloudFront
// distributions and Lambda@Edge.
//
// See also GlobalProvider, which is for all the other global services that may
// be configured anywhere.
func UsEast1Provider(roleArn string) Provider {
	return Provider{
		Alias:       "us-east-1",
		Region:      "us-east-1",
		RoleArn:     roleArn,
		SessionName: "Terraform",
	}
}

func (p Provider) Ref() Value {
	return Uf("aws.%s", p.Region)
}

func (Provider) Template() string {
	return fmt.Sprintf(`provider "aws" {
{{- if .Alias}}
	alias = "{{.Alias}}"
{{- end}}
{{- if .RoleArn}}
{{- if .SessionName}}
	assume_role {
{{- if .ExternalId}}
		external_id = "{{.ExternalId}}"
{{- end}}
		role_arn = "{{.RoleArn}}"
		session_name = "{{.SessionName}}"
	}
{{- end}}
{{- end}}
  default_tags {
    tags = {
      Manager          = "Terraform"
      SubstrateVersion = "%s"
    }
  }
{{- if .Region}}
	region = "{{.Region}}"
{{- end}}
}`, version.Version)
}

type ProviderAlias string

const (
	DefaultProviderAlias = ProviderAlias("aws")
	NetworkProviderAlias = ProviderAlias("aws.network")
)

func ProviderAliasFor(region string) ProviderAlias {
	return ProviderAlias(fmt.Sprintf("aws.%s", region))
}
