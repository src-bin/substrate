# Adding administrators to your AWS organization

The Administrator role in all your Substrate-managed AWS accounts is managed outside of Terraform because Substrate configures Terraform to assume the Administrator role during Terraform runs. A classic chicken-and-egg problem. By default, the Administrator role in your Substrate account can assume the Administrator role in all your service accounts and human users with the `RoleName` attribute set to “Administrator” in your identity provider can assume the Administrator role in your Substrate account. And while that's very often enough, there are plenty of reasons you might want to extend this.

You may have additional human users that can't, for whatever reason, be granted accounts in your identity provider. You may choose to grant these folks access via an IAM user (with two-factor authentication, of course!), via a separate AWS account to which they already have access, or by using Terraform to configure a parallel identity provider they can access. In all of these cases, these users will need to assume the Administrator role.

You similarly may wish to grant a third-party SaaS CI/CD product access to run `terraform apply` (or `substate account create|update`) on your behalf. If you're using GitHub Actions, you can follow their guide to [configuring OpenID Connect in Amazon Web Services](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services) to arrange for GitHub Actions to assume roles in your AWS accounts without managing an IAM user (which can't have two-factor authentication!) and handing long-lived and very privileged access key to a third party. If you're not using GitHub Actions and the third party can't be configured to assume a role then you might have to create an IAM user and access key. In all of these cases, too, these principals will need to assume the Administrator role.

To allow additional principals, whether IAM user or roles and whether they're coming from AWS accounts you control or don't, and regardless of whether they came directly or via an integration with another identity provider, you can populate the `substrate.Administrator.assume-role-policy.json` file in your Substrate repository.

The contents of `substrate.Administrator.assume-role-policy.json` must be a well-formed JSON document in the structure of an AWS IAM assume role policy. Statements in this file must specify `"Action"` as one or more of the `sts:AssumeRole` family of actions, must _not_ specify a `"Resource"`, and must specify one or more `"Principal"` identifiers (most likely IAM user or role ARNs).

The simplest `substrate.Administrator.assume-role-policy.json` looks like this:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::<account-number>:role/<RoleName>"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

You can include as many principals as you'd like in the innermost list.

Every time you update this file, you'll need to re-run `substrate setup` in order to update the Administrator role in all the relevant accounts. This policy will be merged with the policy Substrate generates for the Administrator role (since roles may only have a single assume-role policy).

Once successfully applied, your additional administrators will be able to assume the Administrator role in all your accounts.

Note, too, that this pattern can be applied to the Auditor role using the `substrate.Auditor.assume-role-policy.json` file per [auditing your Substrate-managed AWS organization](../compliance/auditing.html).

[Adding custom IAM roles for humans or services](custom-iam-roles.html) offers far more flexible tools for managing IAM roles.
