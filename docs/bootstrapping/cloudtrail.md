# Configuring AWS CloudTrail

If you don't already have CloudTrail configured in your organization, you'll want to get that setup. The first trail in any organization is free (though subsequent trails can be quite pricey, so beware) and provides an enormously valuable audit trail of everything that's happening in your organization.

Run:

```shell-session
substrate setup cloudtrail
```

This program finds or creates the audit account and enables CloudTrail to log everything to a locked-down S3 bucket in that account.
