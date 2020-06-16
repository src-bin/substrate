package terraform

// managed by go generate; do not edit by hand

func terraformBackendTemplate() string {
	return `# managed by Substrate; do not edit by hand

/*
terraform {
  backend "s3" {
    bucket         = "{{.Bucket}}"
    dynamodb_table = "{{.DynamoDBTable}}"
    key            = "{{.Key}}"
    region         = "{{.Region}}"
  }
}
*/
`
}
