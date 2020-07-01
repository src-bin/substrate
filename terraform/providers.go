package terraform

import (
	"fmt"
)

type Provider struct {
	Alias, AliasPrefix, AliasSuffix, Region string // if unset, Alias is constructed from the other three
	RoleArn                                 string
	SessionName, ExternalId                 string
}

// GlobalProvider returns a Terraform provider that assumes the Administrator
// role in sess's account in us-east-1.  It's functionally equivalent to
// ProviderFor(sess, "us-east-1") but sets the provider's alias to "global"
// so it may be easily distinguished.  It chooses us-east-1 to accommodate
// global services like Lambda@Edge which, in addition to being global, may
// only be configured in us-east-1.
func GlobalProvider(roleArn string) Provider {
	return Provider{
		Alias:       "global",
		Region:      "us-east-1",
		RoleArn:     roleArn,
		SessionName: "Terraform",
	}
}

// NetworkProviderFor returns a Terraform provider for discovering the VPCs and
// subnets in the given region's network.  The given sess must be in the master
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

func (p Provider) Ref() Value {
	return Uf("aws.%s", p.Region)
}

func (Provider) Template() string {
	return `provider "aws" {
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
{{- if .Region}}
	region = "{{.Region}}"
{{- end}}
}`
}

type ProviderAlias string

const (
	DefaultProviderAlias = ProviderAlias("aws")
	GlobalProviderAlias  = ProviderAlias("aws.global")
	NetworkProviderAlias = ProviderAlias("aws.network")
)

func ProviderAliasFor(region string) ProviderAlias {
	return ProviderAlias(fmt.Sprintf("aws.%s", region))
}
