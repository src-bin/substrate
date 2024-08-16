# Adding custom IAM roles for humans or services

Once you've isolated your various environments and services in their own AWS accounts, you'll no doubt need to get into those accounts to operate those services, deploy changes, and debug unexpected behavior. Substrate provides the Administrator and Auditor roles automatically but in many cases Administrator, which is allowed to use all AWS APIs in all your AWS accounts, is too privileged and Auditor, which has limited read-only access to all your AWS accounts, is too restricted.

If Administrator is too privileged and Auditor is too restricted, you need to create custom IAM roles. Substrate manages IAM roles for cross-account access better than anything else around.

Note, too, that IAM roles created by `substrate role create` are meticulously tagged and tracked to enable [enumerating all your custom AWS IAM roles](../access/enumerating-all-your-aws-iam-roles.md), complete with the high-level account selections that control where each role exists, the parameters of each one's assume-role policy, and the policies that are attached to allow access to AWS APIs.

## Assigning a custom IAM role to humans in your identity provider

```shell-session
substrate role create --role <RoleName> [account selection flags] --humans [policy attachment flags]
```

Once you've created an IAM role for humans, you need to assign it to some humans in your identity provider. Set the AWS/RoleName attribute to the name of the custom IAM role.

## Limiting access to certain accounts

```shell-session
substrate role create --role <RoleName> --domain <domain> --all-environments [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --all-domains --environment <environment> [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --management [assume-role-policy flags] [policy attachment flags]
```

See `substrate role create --help` for a complete description of all the account selection flags and how they may be combined.

It's common to specify `--administrator-access` in these situations, following the principle of granting very broad access _within_ an AWS account but very restricted access between AWS accounts or indeed to other, unrelated AWS accounts.

## Allowing access only to certain AWS APIs

AWS provide managed IAM policies for the vast majority of their services, which can be attached to custom IAM roles:

```shell-session
substrate role create --role <RoleName> [account selection flags] [assume-role-policy flags] --policy-arn <managed-policy-ARN> --policy-arn <another-managed-policy-ARN>
```

This technique is very useful when creating a custom IAM role for a finance team or another team with very specific needs across your entire organization.

## Iterating on your custom IAM role

There's no need to fret about getting your role definitions exactly right on the first try, as you can start small, in a single account, and iterate until your custom IAM role is ready to be created in lots or even all of your AWS accounts. You might follow a progression like this, assuming you have environments called “development”, “staging”, and “production”:

```shell-session
substrate role create --role <RoleName> --domain <domain> --environment development [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --domain <domain> --environment development --environment staging [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --all-domains --environment development [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --all-domains --environment development --environment staging [assume-role-policy flags] [policy attachment flags]
substrate role create --role <RoleName> --all-domains --all-environments [assume-role-policy flags] [policy attachment flags]
```

## Referencing custom IAM roles in Terraform

Even as expressive as `substrate role create` is, you may find reason to take a role created through this command and manage what it's allowed to do in Terraform. The most common reason to do this is when you want to allow the role to do more in e.g. your development environment than in your production environment.

First, create a role using `substrate role create` but don't specify any policy attachment flags (i.e. don't specify `--administrator-access`, `--read-only-access`, `--policy-arn`, or `--policy`).

Then, in a file in `modules/<domain>/global` or `modules/common/global`, include Terraform code like this:

```terraform
data "aws_iam_role" "<custom_role_name>" {
  name = "<CustomRoleName>"
}

resource "aws_iam_role_policy_attachment" "<custom_role_policy>" {
  policy_arn = <policy_arn>
  role       = aws_iam_role.<custom_role_name>.name
}
```

The sky's the limit on how custom your custom IAM roles can be once you take control in Terraform.
