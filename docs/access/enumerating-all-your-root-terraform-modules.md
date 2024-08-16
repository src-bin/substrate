# Enumerating all your root Terraform modules

`substrate terraform root-modules` prints the relative pathname to all your root Terraform modules to standard output. Their order is as follows:

1. Deploy account, global before regional
2. Network account, global before regional
3. Network peering, in order of `substrate.environments` and alphabetical order of regions
4. Admin accounts, in order of `substrate.qualities`
5. Service accounts, in alphabetical order of domain, then in order of `substrate.environments`, then in order of `substrate.qualities`

You can use this list to generate CI/CD configurations that run Terraform directly, though do consider using `substrate account list --format shell` instead to run Substrate, too.
