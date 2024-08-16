# Changes to Substrate commands in 2024.01

| Syntax in 2023.12 and earlier | Syntax in 2024.01 and later | Notes |
|-------------------------------|-----------------------------|-------|
| `substrate --shell-completion` | Bash and Z shell: `. <(substrate shell-completion)`<br>Fish: `. $(substrate shell-completion \| psub)` | Please see [Configuring Substrate shell completion](bootstrapping/shell-completion.html) and update your shell configuration. |
| `substrate accounts` | `substrate account list` | |
| `substrate create-account` | `substrate account adopt`<br>`substrate account create`<br>`substrate account update` | The three forms of `substrate create-account` have been split into their own commands for clarity. `substrate account create` will create and configure the account but exit with an error if the account already exists. `substrate account adopt` will bring an existing account under Substrate's management. `substrate account update` ensures an existing Substrate-managed account is properly configured and then runs Terraform as `substrate create-account` used to do. |
| `substrate roles` | `substrate role list` | |
| `substrate create-role` | `substrate role create` | |
| `substrate delete-role` | `substrate role delete` | |
| `substrate terraform` | `substrate terraform install` | |
| | `substrate terraform` | This new form of `substrate terraform` brings `--domain` (or `-d`) and `--environment` (or `-e`) to Terraform to simplify selecting the directory in which to run Terraform. For example, `substrate terraform --domain www --environment staging --region us-west-2 plan` is the same as `terraform -chdir=root-modules/www/staging/default/us-west-2 plan`. Bonus: Substrate's autocomplete works for `--domain`, `--environment`, `--region`, etc. and then gives way to Terraform's autocomplete for `init`, `plan`, `apply`, etc. |
| `substrate root-modules` | `substrate terraform root-modules` | |
| `substrate setup-cloudtrail` | `substrate setup cloudtrail` | |
| `substrate setup-debugger` | `substrate setup debugger` | |
| `substrate delete-static-access-keys` | `substrate setup delete-static-access-keys` | |

In addition to these changes, Substrate 2024.01 switched to the POSIX standard of using two dashes to prefix long option names, i.e. `-domain` becomes `--domain`, etc.

Pass the `--help` option to any Substrate command to see the full details.
