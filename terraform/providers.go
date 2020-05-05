package terraform

import "github.com/src-bin/substrate/awsutil"

type Provider struct {
	AccountId, RoleName, SessionName, ExternalId string
	//Alias string
	Region string
}

func (p Provider) AllRegions() Blocks {
	blocks := NewBlocks()
	for _, region := range awsutil.Regions() {
		p.Region = region
		blocks.Push(p)
	}
	return blocks
}

func (Provider) Template() string {
	return `provider "aws" {
	alias = "{{.Region}}"
	assume_role {
		role_arn     = "arn:aws:iam::{{.AccountId}}:role/{{.RoleName}}"
		session_name = "{{.SessionName}}"
{{if .ExternalId -}}
		external_id  = "{{.ExternalId}}"
{{end -}}
	}
	region = "{{.Region}}"
}`
}
