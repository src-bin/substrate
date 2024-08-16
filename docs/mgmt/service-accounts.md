# Managing your infrastructure in service accounts

With your identity provider integrated, you're now entering the choose-your-own adventure phase of adopting Substrate. You'll use service accounts to host the meat of your infrastructure — potentially lots of them.

Substrate helps you use multiple accounts to separate environments (like _development_ or _production_). Substrate also encourages you to organize your services into domains — single services or groups of tightly-coupled services — that help you reduce the blast radius of changes. You should read about [domains, environments, and qualities](../ref/domains-environments-qualities.md) to get a feel for it.

You've probably got in mind the first thing you're going to build with Substrate's help, so next you can jump straight into [adding a domain](adding-a-domain.md). Or just run a command like this:

```shell-session
substrate account create --domain <domain> --environment <environment>
```

After creating your first (or your fiftieth) service account, you'll be [writing Terraform code](writing-terraform-code.md). Substrate doesn't change or restrict how you write Terraform code but it does introduce a few handy shortcuts you can use to name things sensibly and get access to the network it's configured for each of your environments. To run your Terraform code, use `substrate account update`, `substrate terraform`, or `terraform` itself.

The [Substrate documentation](../) has architectural reference material, day-to-day advice, runbooks for emergencies, and more.
