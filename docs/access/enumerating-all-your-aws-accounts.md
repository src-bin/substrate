# Enumerating all your AWS accounts

Substrate will maintain `substrate.accounts.txt` in your Substrate repository as you create new admin and service accounts, providing a reference that's close at hand and even committed to version control (in case AWS is well and truly broken). The `substrate account list` command updates that file and then prints it out. But it accepts a `-format` option, too.

`substrate account list --format json` makes it easy to program against your list of accounts. It's equivalent to the `organizations:ListAccounts` API with each account decorated with its tags, making domain, environment, and quality accessible, too.

`substrate account list --format shell` prints a shell program that will run the appropriate Substrate command against every account in your organization. This is mighty convenient during Substrate upgrades or in CI/CD systems, especially when you add the `--no-apply` or `--auto-approve` options (which influence how Terraform is eventually invoked).
