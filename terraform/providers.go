package terraform

import (
	"fmt"

	"github.com/src-bin/substrate/regions"
)

type Provider struct {
	Alias, AliasPrefix, AliasSuffix, Region string // if unset, Alias is constructed from the other thre
	AccountId, RoleName                     string // for constructing the role ARN; possibly should just be RoleArn
	SessionName, ExternalId                 string
}

// AllRegions creates a provider block for every AWS region.  It purposely
// includes regions we're avoiding because if a region is added to that list
// after resource blocks that reference it are added to Terraform, the
// provider will be necessary in order to destroy those resources.
func (p Provider) AllRegions() *File {
	file := NewFile()
	for _, region := range regions.All() {
		p.Region = region
		file.Push(p)
	}
	return file
}

// AllRegionsAndGlobal does everything AllRegions does plus adds a provider
// called aws.global in us-east-1 to be used by services which are global, like
// CloudFront or IAM, or rooted in us-east-1, like Lambda@Edge.
func (p Provider) AllRegionsAndGlobal() *File {
	file := p.AllRegions()
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
{{- if .Alias}}
	alias = "{{.Alias}}"
{{- else}}
	alias = "{{if .AliasPrefix}}{{.AliasPrefix}}-{{end}}{{.Region}}{{if .AliasSuffix}}-{{.AliasSuffix}}{{end}}"
{{- end}}
{{- if .AccountId}}
{{- if .RoleName}}
{{- if .SessionName}}
	assume_role {
{{- if .ExternalId}}
		external_id  = "{{.ExternalId}}"
{{- end}}
		role_arn     = "arn:aws:iam::{{.AccountId}}:role/{{.RoleName}}"
		session_name = "{{.SessionName}}"
	}
{{- end}}
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
