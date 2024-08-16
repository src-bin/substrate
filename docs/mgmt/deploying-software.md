# Deploying software

Substrate tries hard not to constrain your architecture where there viable options aplenty. Deploying software is an activity that falls into this category, yet there are a few things that Substrate does to make it easy to go in whatever direction you so choose.

The first and most basic thing Substrate does is create its deploy account. When you use separate AWS accounts for every environment, you need someplace that can host your deployable artifacts as they're developed, built, tested, and promoted through your environments on their way to serving your customers. The deploy account is meant to be that place.

## S3

A great many organizations use Amazon S3 as a repository for software artifacts - tarballs, Debian packages, RPMs, etc. - and Substrate takes the liberty of creating S3 buckets for you to use to distribute these artifacts. The buckets named `<prefix>-deploy-artifacts-<region>` (substituting your chosen prefix as stored in `substrate.prefix` and the name of each region you're using) are ready and waiting for whatever you want to store there (but be sure to set the `bucket-owner-full-control` canned ACL). Every account in your organization has access to read and write objects in these versioned buckets. Where you go from there is up to you.

## ECR

Some organizations use higher-level repositories, too, as offered by AWS ECR. We recommend these repositories be created in your deploy account by, for example, placing the following in `modules/deploy/regional/main.tf`:

```
data "aws_iam_policy_document" "organization" {
  statement {
    actions = local.actions
    condition {
      test     = "StringEquals"
      variable = "aws:PrincipalOrgID"
      values   = [data.aws_organizations_organization.current.id]
    }
    principals {
      identifiers = ["*"]
      type        = "AWS"
    }
  }
  statement {
    actions = local.actions
    principals {
      identifiers = ["codebuild.amazonaws.com"]
      type        = "Service"
    }
  }
}

data "aws_organizations_organization" "current" {}

locals {
  actions = [
    "ecr:BatchCheckLayerAvailability",
    "ecr:BatchGetImage",
    "ecr:CompleteLayerUpload",
    "ecr:CreateRepository",
    "ecr:DescribeImages",
    "ecr:DescribeImageScanFindings",
    "ecr:DescribeRepositories",
    "ecr:GetAuthorizationToken",
    "ecr:GetDownloadUrlForLayer",
    "ecr:GetLifecyclePolicy",
    "ecr:GetRepositoryPolicy",
    "ecr:InitiateLayerUpload",
    "ecr:ListImages",
    "ecr:ListTagsForResource",
    "ecr:PutImage",
    "ecr:PutImageScanningConfiguration",
    "ecr:PutImageTagMutability",
    "ecr:StartImageScan",
    "ecr:TagResource",
    "ecr:UntagResource",
    "ecr:UploadLayerPart",
  ]
  repos = [
    "usage-event-aggregator",
    "usage-event-deduplicator",
  ]
}

resource "aws_ecr_repository" "hello-world" {
  #image_tag_mutability = "IMMUTABLE" # desirable but potentially annoying
  name = "hello-world"
  tags = {
    Manager = "Terraform"
    Name    = "hello-world"
  }
}
```

Once this is implemented, all your AWS accounts (and no one else's) will be authorized to push and pull Docker containers to and from your ECR repository.

If you use raw EC2 and build your own AMIs, then you'll perhaps be interested in sharing those AMIs with other accounts in your AWS organization. See [Sharing an AMI with specific AWS accounts](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/sharingamis-explicit.html) for more information.
