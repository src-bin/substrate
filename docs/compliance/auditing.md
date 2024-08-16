# Auditing your Substrate-managed AWS organization

One of the very first things Substrate does is to configure AWS CloudTrail with a single trail that covers all organization accounts in all regions. To access this wealth of data, assume the `Auditor` role in your audit account.

On the command line: `substrate assume-role --special audit`

Or you can assume the role in the AWS Console via your Intranet's Accounts page.

In either case, the data you seek is in the `<prefix>-cloudtrail` (substituting your chosen prefix as stored in `substrate.prefix`) S3 bucket. You can download it to analyze locally or [query it with Amazon Athena](https://docs.aws.amazon.com/athena/latest/ug/cloudtrail-logs.html). The most straightforward way to proceed is by creating Athena tables for each AWS account; partition projection to cover the entire organization is unsolved.

## Allowing third parties to audit your Substrate-managed AWS organization

Many tools have grown the ability to assume an IAM role in your AWS account to perform some auditing or monitoring feature on your behalf. To allow these into your AWS organization, create an assume role policy in `substrate.Auditor.assume-role-policy.json` with contents like this:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::ACCOUNT_NUMBER:role/ROLE_NAME"
        ]
      }
    }
  ]
}
```

You can include as many principals as you'd like in the innermost list.

Every time you update this file, you'll need to re-run `substrate setup` and `substrate account update` for each of your service accounts.

This will authorize third-party principals to `sts:AssumeRole` using the ARN of one of your Auditor roles and operate as you would there.

Note, too, that this pattern can be applied to the Administrator role using the `substrate.Administrator.assume-role-policy.json` file per [adding administrators to your AWS organization](../mgmt/adding-administrators.html).

If allowing the third party organization-wide read-only access is too permissive, consider using `substrate role create` to target certain accounts and grant exactly the permissions the third party requires.
