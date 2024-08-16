# Finishing up in your management account

When `substrate setup` exits, you should add `.substrate.*` to your version control system's ignore list (e.g. `.gitignore`) and commit the rest of the files Substrate generated to version control.

## Fetch temporary AWS credentials

Test out your shiny new integration between AWS and your identity provider by fetching some temporary AWS credentials to use today and to learn the command you can use to get new credentials tomorrow:

```shell-session
eval $(substrate credentials)
```

With this working, we can tidy up your management account.

## Deleting unnecessary root access keys

As a final test before deleting your root access key, verify that you can run `substrate assume-role --management`. If so, you can finally delete your root and OrganizationAdministrator access keys. They're simply security liabilities. Let's delete them:

1. Run `substrate setup delete-static-access-keys` to delete access keys for the Substrate IAM user in your management account
2. Visit [https://console.aws.amazon.com/iam/home#/security\_credentials](https://console.aws.amazon.com/iam/home#/security\_credentials) while signed in using the root email address, password, and second factor on your management account
3. Scroll to the _Access keys_ section
4. Select your root access key
5. Click **Actions**
6. Click **Delete**
7. Click **Deactivate**
8. Paste the access key ID into the confirmation prompt
9. Click **Delete**

From now on, the Credential and Instance Factories are how you access your organization via the command line.

## Delegate access to billing data

While you're logged into your management account using the root credentials, follow these steps to delegate access to billing data to people and tools assuming IAM roles.

1. Visit [https://console.aws.amazon.com/billing/home?#/account](https://console.aws.amazon.com/billing/home?#/account)
2. Open the _IAM User and Role Access to Billing Information_ section
3. Check “Activate IAM Access”
4. Click **Update**
5. Visit [https://console.aws.amazon.com/billing/home#/costexplorer](https://console.aws.amazon.com/billing/home#/costexplorer)
6. Click **Enable Cost Explorer** or **Launch Cost Explorer** (whichever is displayed)
