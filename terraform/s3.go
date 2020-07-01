package terraform

type S3Bucket struct {
	Bucket   Value
	Label    Value
	Policy   Value
	Provider ProviderAlias
	Tags     Tags
}

func (b S3Bucket) Ref() Value {
	return Uf("aws_s3_bucket.%s", b.Label)
}

func (S3Bucket) Template() string {
	return `resource "aws_s3_bucket" {{.Label.Value}} {
  bucket   = {{.Bucket.Value}}
{{- if .Policy}}
  policy   = {{.Policy.Value}}
{{- end}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags     = {{.Tags.Value}}
  versioning {
    enabled = true
  }
}
resource "aws_s3_bucket_public_access_block" {{.Label.Value}} {
  block_public_acls       = true
  block_public_policy     = true
  bucket                  = {{.Ref.Value}}.bucket
  ignore_public_acls      = true
{{- if .Provider}}
  provider                = {{.Provider}}
{{- end}}
  restrict_public_buckets = true
}`
}
