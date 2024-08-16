# Using Amazon EC2 when IMDSv2 is required

Substrate configures your AWS organization to prevent you from using Amazon EC2 in a way that's vulnerable to server-side request forgery attacks against the instance metadata service (IMDS, which you probably better know as 169.254.169.254) by requiring you to use IMDSv2. All of the AWS SDKs have long used this new method of fetching instance metadata including temporary IAM role credentials, instance identity documents, and the instance's public SSH key.

The rest of this page offers pointers on using EC2 in the face of this important security measure.

## Instance Factory

Every instance provisioned by your Instance Factory requires the use of IMDSv2 without you doing anything special.

## AWS Console

When using the AWS Console to launch an instance, almost every option remains open to however you want to configure it. There is one exception. In the launch wizard:

1. Scroll down to the bottom of the page
2. Click **Advanced details** to reveal the rest of the options
3. Change _Metadata version_ to “V2 only (token required)”
4. Click **Launch instance**

## AWS CLI

If you're launching EC2 instances directly using the AWS CLI, provide the `--metadata-options` option thus:

```shell-session
aws ec2 run-instances ... --metadata-options HttpEndpoint=enabled,HttpProtocolIpv6=enabled,HttpTokens=required,InstanceMetadataTags=enabled ...
```

## EC2 API

Similarly, if you're calling the EC2 API directly, include the following key and value in the request body:

```json
"MetadataOptions": {
    "HttpEndpoint":"enabled",
    "HttpProtocolIpv6":"enabled",
    "HttpTokens":"required",
    "InstanceMetadataTags":"enabled"
}
```

If you're using a language SDK (as you most likely are), translate this structure into your language's syntax and provide it alongside the rest of the options to `ec2:RunInstances`.

## EC2 via Terraform

Almost the same structure applies if you're using Terraform's `aws_instance` resource (though for reasons unknown it doesn't expose the IPv6 option yet):

```
resource "aws_instance" "example" {
  # ...
  metadata_options {
    http_endpoint          = "enabled"
    #http_protocol_ipv6     = "enabled" # Terraform doesn't support this for EC2 yet
    http_tokens            = "required"
    instance_metadata_tags = "enabled"
  }
  # ...
}
```

## EC2 Launch Templates API

If you're creating a launch template, ensure that it sets the same `MetadataOptions` in the EC2 API as the prior strategies:

```json
"MetadataOptions": {
    "HttpEndpoint":"enabled",
    "HttpProtocolIpv6":"enabled",
    "HttpTokens":"required",
    "InstanceMetadataTags":"enabled"
}
```

## EC2 Launch Templates via Terraform

And, no surprise, you can specify the same options in Terraform (including the IPv6 option):

```
resource "aws_launch_template" "example" {
  # ...
  metadata_options {
    http_endpoint          = "enabled"
    http_protocol_ipv6     = "enabled"
    http_tokens            = "required"
    instance_metadata_tags = "enabled"
  }
  # ...
}
```
