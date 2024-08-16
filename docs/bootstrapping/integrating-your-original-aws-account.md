# Integrating your original AWS account(s)

Most likely, you've already opened (at least) one AWS account(s), and you're probably a little apprehensive about what will happen to it once you adopt Substrate. Let's address the two most common concerns right away:

1. Don't worry, you won't be left maintaining two AWS worlds forever.
2. Yes, you'll be able to keep your AWS credit balance.

In fact, integrating your original AWS account(s) into your new Substrate-managed AWS organization is a good idea for a few reasons:

* It can quickly improve your security posture if you create an Administrator role there that can be assumed by the Administrator role in your Substrate account, thus allowing you to delete long-lived access keys and uncontrolled IAM users there.
* It may simplify some migrations by allowing policies written using the `aws:PrincipalOrgID` condition key to interoperate with this account. (Don't worry if policies and condition keys are not familiar topics.)
* Integrating access to its billing data will give you better visibility into where you're spending money with AWS.

You can invite as many AWS accounts as you like into your new Substrate-managed AWS organization but only if you have _not_ configured AWS Organizations manually in those accounts. If you have and you would still like to adopt Substrate, ask us at [hello@src-bin.com](mailto:hello@src-bin.com) and we'll work with you to make everything right.

## Inviting the account into your organization

To begin, you'll need root (not IAM user) login credentials for the AWS Console for the account you wish to invite into your organization. If you don't have them now or you just don't want to do this now, feel free to skip this section and come back to it later. When you're ready, proceed:

1. Open the Accounts page of your Intranet
2. Assume the OrganizationAdministrator role in your management account
3. Visit [https://console.aws.amazon.com/organizations/home?#/accounts](https://console.aws.amazon.com/organizations/home?#/accounts)
4. Click **Add account**
5. Click **Invite account**
6. Enter the email address of your original AWS account, the one that you're inviting into your organization (or its account number, if you have that handy)
7. Click **Invite**
8. In an incognito window, sign into the AWS Console using root (not IAM user) credentials for the account you just invited into your organization
9. In that incognito window, visit [https://console.aws.amazon.com/cloudtrail/home#/dashboard](https://console.aws.amazon.com/cloudtrail/home#/dashboard) and delete any existing trails (to avoid very expensive surprises in your consolidated AWS bills)
10. In that incognito window, visit [https://console.aws.amazon.com/organizations/home#/invites](https://console.aws.amazon.com/organizations/home#/invites)
11. Click **Accept**
12. Click **Confirm**

If you stop here, you'll have integrated billing data from your original AWS account into your organization and you'll have the ability to constrain your original AWS account using service control policies.

To allow your Substrate account to access this original AWS account, use the AWS Console in your original AWS account to create the Administrator role as follows:

1. Note the Administrator role ARN in the table listing your Substrate account in `substrate.accounts.txt`
2. Create a new role named Administrator in your original AWS account with the following assume role policy, substituting your Substrate account number:

    ```json
     {
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Principal": {
             "AWS": [
               "arn:aws:iam::<Substrate-account-number>:role/Administrator",
               "arn:aws:iam::<Substrate-account-number>:role/Substrate",
               "arn:aws:iam::<Substrate-account-number>:user/Substrate",
               "arn:aws:iam::<management-account-number>:role/Substrate",
               "arn:aws:iam::<management-account-number>:user/Substrate"
             ]
           },
           "Action": "sts:AssumeRole"
         }
       ]
     }
    ```
3. Attach the AWS-managed AdministratorAccess policy to this new role

This manual change in the AWS Console, which would usually be distasteful, has paved the way for Substrate to manage your original AWS account, especially this Administrator role. To complete the integration, run `substrate account adopt --domain <domain> --environment <environment> --number <12-digit-account-number>` to tag your original AWS account, manage its Administrator and Auditor roles, and generate its basic Terraform directory structure.

After you've completed these steps, your original AWS account is part of your Substrate-managed AWS organization, just like any other service account.
