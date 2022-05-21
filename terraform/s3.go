package terraform

const (
	BucketOwnerPreferred = "BucketOwnerPreferred"
	ObjectWriter         = "ObjectWriter"
)

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
  bucket = {{.Bucket.Value}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
}

{{- if .Policy}}
resource "aws_s3_bucket_policy" {{.Label.Value}} {
  bucket = {{.Ref.Value}}.bucket
  policy = {{.Policy.Value}}
}

{{- end}}
resource "aws_s3_bucket_public_access_block" {{.Label.Value}} {
  block_public_acls = true
  block_public_policy = true
  bucket = {{.Ref.Value}}.bucket
  ignore_public_acls = true
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" {{.Label.Value}} {
  bucket = {{.Ref.Value}}.bucket
  versioning_configuration {
    status = "Enabled"
  }
}`
}

type S3BucketOwnershipControls struct {
	Bucket          Value
	Label           Value
	ObjectOwnership Value
	Provider        ProviderAlias
}

func (boc S3BucketOwnershipControls) Ref() Value {
	return Uf("aws_s3_bucket_ownership_controls.%s", boc.Label)
}

func (S3BucketOwnershipControls) Template() string {
	return `resource "aws_s3_bucket_ownership_controls" {{.Label.Value}} {
  bucket = {{.Bucket.Value}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  rule {
    object_ownership = {{.ObjectOwnership.Value}}
  }
}`
}
