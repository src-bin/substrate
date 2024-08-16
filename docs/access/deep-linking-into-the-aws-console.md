# Deep-linking into the AWS Console

As of Substrate 2022.09, you can deep-link into the AWS Console via your Intranet. You should try to take this extra step whenever you're presenting your coworkers with links into the AWS Console to avoid two points of friction:

1. They might not be logged in and the AWS Console isn't very aware of outside identity providers like the ones you integrate via Substrate.
2. They might be logged into a different AWS account, which will almost certainly result in something akin to a 404 Not Found within the AWS Console.

By deep-linking via your Intranet, you can ensure folks will be logged in and to the appropriate AWS account.

You must construct a URL like this:

<code>https://<em>example.com</em>/accounts?next=https%3A%2F%2Fconsole.aws.amazon.com%2F<em>some-service%2Fsome-resource%3Fsome-querystring</em>&number=<em>12-digit-AWS-account-number</em>&role=<em>role-name</em></code>

* <code><em>example.com</em></code>: As is the convention in this documentation, replace _example.com_ with your Intranet's DNS domain name.
* <code>https%3A%2F%2Fconsole.aws.amazon.com%2F<em>some-service%2Fsome-resource%3Fsome-querystring</em></code>: The `next` query parameter must contain a URL-encoded URL to a page on `console.aws.amazon.com` or a subdomain (which means that, yes, you can link to regional pages in the AWS Console).
* <code><em>12-digit-AWS-account-number</em></code>: The `number` query parameter must contain the 12-digit AWS account number of the account that owns the resource(s) you're linking to.
* <code><em>role-name</em></code>: The `role` query parameter must contain the name of an IAM role in that account. &ldquo;Auditor&rdquo; is a good choice here because most folks will be able to assume that role, which makes it reasonable to e.g. share the deep link in Slack, and because the resulting AWS Console session will be read-only, which makes it a little safer.
