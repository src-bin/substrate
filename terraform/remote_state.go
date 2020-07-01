package terraform

type RemoteState struct {
	Config   RemoteStateConfig
	Label    Value
	Provider ProviderAlias
}

type RemoteStateConfig struct {
	Bucket, DynamoDBTable, Key, Region, RoleArn string
}

func (rs RemoteState) Ref() Value {
	return Uf("data.terraform_remote_state.%s", rs.Label)
}

func (RemoteState) Template() string {
	return `data "terraform_remote_state" {{.Label.Value}} {
  backend = "s3"
  config = {
    bucket = "{{.Config.Bucket}}"
    dynamodb_table = "{{.Config.DynamoDBTable}}"
    key = "{{.Config.Key}}"
    region = "{{.Config.Region}}"
    role_arn = "{{.Config.RoleArn}}"
  }
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
}`
}
