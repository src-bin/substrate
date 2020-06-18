# managed by Substrate; do not edit by hand

terraform {
  backend "s3" {
    bucket         = "{{.Bucket}}"
    dynamodb_table = "{{.DynamoDBTable}}"
    key            = "{{.Key}}"
    region         = "{{.Region}}"
    role_arn       = "{{.RoleArn}}"
  }
}
