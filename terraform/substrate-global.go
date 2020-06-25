package terraform

// managed by go generate; do not edit by hand

func substrateGlobalTemplate() map[string]string {
	return map[string]string{
		"providers.tf": `provider "aws" { alias = "global" }
`,
		"outputs.tf":   `output "tags" {
  value = {
    domain      = data.external.tags.result.Domain
    environment = data.external.tags.result.Environment
    quality     = data.external.tags.result.Quality
  }
}
`,
		"tags.tf":      "data \"aws_caller_identity\" \"current\" {}\n\ndata \"external\" \"tags\" {\n  program = [\n    \"substrate-assume-role\", \"-master\", \"-quiet\", \"-role=OrganizationReader\",\n    \"aws\", \"organizations\", \"list-tags-for-resource\",\n    \"--resource-id\", data.aws_caller_identity.current.account_id,\n    \"--query\", \"{Domain:Tags[?Key==`Domain`].Value|[0],Environment:Tags[?Key==`Environment`].Value|[0],Quality:Tags[?Key==`Quality`].Value|[0]}\",\n  ]\n}\n",
	}
}
