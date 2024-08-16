# Moving between AWS accounts

There isn't usually much to do in your Substrate account. Instead, you'll be assuming roles in other accounts where your real work happens. There are three ways this happens:

```shell-session
substrate assume-role
```

Ad-hoc movement throughout your organization is made easy by the `substrate assume-role` command. It understands the layout and can convert domains, environments, and qualities into the appropriate AWS account numbers for you.

To get temporary credentials in your _example development default_ account (once you've created such an account), you'd run `substrate assume-role --domain example --environment development --quality default`. Without any additional arguments, `substrate assume-role` prints shell environment variables so you should wrap it in `eval`. It will also feed the shell an `unassume-role` alias you can use to pop back into your Substrate account:

```sh
eval $(substrate assume-role --domain example --environment development --quality default)
# do whatever you like
unassume-role
```

If you have a specific command you need to run, tack it onto the end thus:

```shell-session
substrate assume-role --domain example --environment development --quality default aws ec2 describe-security-groups
```

In addition to the forms above that allow specifying a domain, environment, and quality, `substrate assume-role` can select your management account with `--management`, your audit, deploy, or network account with `--special audit`, `--special deploy`, or `--special network`, and your Substrate account with `--substrate`. Or you can go completely off-road and specify any arbitrary AWS account with `--number <number>`.

By default, `substrate assume-role` will carry on with the same role name — Administrator (or OrganizationAdministrator, etc. as appropriate) when you're Administrator, Auditor when you're Auditor, and so on. You can specify a different role name using `--role <role>`.

## Terraform

A lot of work in your AWS organization might happen in Terraform and not ad-hoc shell sessions. `substrate account adopt|create` creates a root Terraform module for you with providers configured to assume the appropriate role so you don't have to think about matching credentials in your environment with directories in which you invoke `terraform apply`. There are a few ways you can invoke Terraform, depending on what you're after:

* `substrate account update --domain <domain> --environment <environment>`: Invoke Terraform on an account's global root module and then on each of its regional root modules.
* `substrate terraform --domain <domain> --environment <environment> init|plan|apply|...`: Invoke Terraform in a particular root module and with credentials for a particular AWS account, specified by Substrate-managed names using `--domain` and `--environment`, `--management`, `--special`, or `--substrate`.
* `terraform -chdir=<root-module> init|plan|apply|...`: Invoke Terraform directly in a particular directory. Its configuration will take care of assuming the appropriate role.

## AWS Console

The AWS Console includes a “switch role” feature that you're welcome to use but [accessing the AWS Console](accessing-the-aws-console.md) shows that you probably won't need it. In your Substrate-managed AWS organization, access to the AWS Console feels less like switching roles and more like going straight into the account you need to access.
