# Integrating your Google identity provider

`substrate setup` will ask for several inputs, which this page will help you provide from your Google identity provider.

These steps must be completed by a Google Super Admin. Be mindful, too, of which Google account you're using if you're signed into more than one in the same browser profile. Google has a habit of switching accounts when you least expect it.

## Create a custom schema for assigning IAM roles to Google users

1. Visit [https://admin.google.com/ac/customschema](https://admin.google.com/ac/customschema) in a browser (or visit [https://admin.google.com](https://admin.google.com), click **Users**, click **More**, and click **Manage custom attributes**)
2. Click **ADD CUSTOM ATTRIBUTE**
3. Enter “AWS” for _Category_
4. Under _Custom fields_, enter “RoleName” for _Name_, select “Text” for _Info type_, select “Visible to user and admin” for _Visibility_, select “Single Value” for _No. of values_
5. Click **ADD**

## Create and configure an OAuth OIDC client

1. Visit [https://console.developers.google.com/](https://console.developers.google.com/) in a browser
2. Click **CREATE PROJECT**
3. Name the project and, optionally, put it in an organization (but don't worry if you can't put it in an organization, because everything still works without one)
4. Click **CREATE**
5. Click **SELECT PROJECT** in the status overlay that appears in the top right corner
6. Click **OAuth consent screen**
7. Select “Internal”
8. Click **CREATE**
9. Enter an _Application name_
10. Select a _User support email_
11. Enter your Intranet DNS domain name in _Authorized domains_
12. In _Developer contact information_, enter one or more _Email addresses_
13. Click **SAVE AND CONTINUE**
14. Click **ADD OR REMOVE SCOPES**
15. Select “.../auth/userinfo.email”, “.../auth/userinfo.profile”, and “openid”
16. Enter “https://www.googleapis.com/auth/admin.directory.user.readonly” in the text input under _Manually add scopes_
17. Click **ADD TO TABLE**
18. Click **UPDATE**
19. Click **SAVE AND CONTINUE**
20. Click **Credentials** in the left column
21. Click **CREATE CREDENTIALS** and then **OAuth client ID** in the expanded menu
22. Select “Web application” for _Application type_
23. Enter a _Name_, if desired
24. Click **ADD URI** in the _Authorized redirect URIs_ section
25. Enter “https://_intranet-dns-domain-name_/login” (substituting your just-purchased or just-transferred Intranet DNS domain name)
26. Click **CREATE**
27. Use the credentials to respond to `substrate setup`'s prompts
28. Click **OK**
29. Visit [https://console.cloud.google.com/apis/library/admin.googleapis.com](https://console.cloud.google.com/apis/library/admin.googleapis.com) in a browser
30. Confirm the project you created a moment ago is selected (its name will be listed next to “Google Cloud Platform” in the header)
31. Click **ENABLE**

## Authorize users to use AWS

1. Visit [https://admin.google.com/ac/users](https://admin.google.com/ac/users) in a browser (or visit [https://admin.google.com](https://admin.google.com) and click **Users**)
2. For every user authorized to use AWS:
   1. Click the user's name
   2. Click **User information**
   3. In the _AWS_ section, click **Add RoleName** and enter the name (not the ARN) of the IAM role they should assume in your Substrate account (“Administrator” for yourself as you're getting started; if for others it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](../../mgmt/custom-iam-roles.html) first)
   4. Click **SAVE**

With your identity provider integrated, jump to [finishing up in your management account](../finishing.html).
