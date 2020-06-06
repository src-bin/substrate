package terraform

import (
	"fmt"

	"github.com/src-bin/substrate/regions"
)

type Provider struct {
	AccountId, RoleName, SessionName, ExternalId string
	Alias                                        string
	Region                                       string
}

// AllRegions creates a provider block for every AWS region.  It purposely
// includes blacklisted regions because if a region is added to the blacklist
// after resource blocks that reference it are added to Terraform, the
// provider will be necessary in order to destroy those resources.
func (p Provider) AllRegions() *File {
	file := NewFile()
	for _, region := range regions.All() {
		p.Alias = region
		p.Region = region
		file.Push(p)
	}

	// Add a provider with a well-known name to use for global services like
	// CloudFront and IAM.  Use us-east-1 becuase services like Lambda@Edge
	// must use that region without exception.
	p.Alias = "global"
	p.Region = "us-east-1"
	file.Push(p)

	return file
}

func (p Provider) Ref() Value {
	return Uf("aws.%s", p.Region)
}

func (Provider) Template() string {
	return `provider "aws" {
	alias = "{{.Alias}}"
	assume_role {
{{- if .ExternalId}}
		external_id  = "{{.ExternalId}}"
{{- end}}
		role_arn     = "arn:aws:iam::{{.AccountId}}:role/{{.RoleName}}"
		session_name = "{{.SessionName}}"
	}
	region = "{{.Region}}"
}`
}

type ProviderAlias string

func GlobalProviderAlias() ProviderAlias {
	return ProviderAlias("aws.global")
}

func ProviderAliasFor(region string) ProviderAlias {
	return ProviderAlias(fmt.Sprintf("aws.%s", region))
}
