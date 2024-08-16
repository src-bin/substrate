# Integrating your Okta identity provider

`substrate setup` will ask for several inputs, which this page will help you provide from your Okta identity provider.

## Create a custom profile attribute

1. Visit your Okta admin panel in a browser
2. Click the **hamburger menu**
3. Click **Profile Editor** in the **Directory** section
4. Click **User (default)** (with type “Okta”)
5. Click **+ Add Attribute**
6. Enter “AWS\_RoleName” for both _Display name_ and _Variable name_
7. Click **Save**

## Create and configure an OAuth OIDC client

1. Visit your Okta admin panel in a browser
2. Click the **hamburger menu**
3. Click **Applications** in the **Applications** section
4. Click **Create App Integration**
5. Select “OAuth - OpenID Connect”
6. Select “Web Application”
7. Click **Next**
8. Customize _App integration name_
9. Change the first/only item in _Sign-in redirect URIs_ to “https://_intranet-dns-domain-name_/login” (substituting your just-purchased or just-transferred Intranet DNS domain name)
10. Remove all _Sign-out redirect URIs_
11. Select “Limit access to selected groups” and select the groups that are authorized to use AWS (or choose another option; this can always be reconfigured)
12. Click **Save**
13. Paste the _Client ID_, _Client secret_, and _Okta domain_ in response to `substrate setup`'s prompts
14. Click **Okta API Scopes**
15. Click **Grant** at the end of the “okta.users.read.self” line

## Authorize users to use AWS

1. Visit your Okta admin panel in a browser
2. Click the **hamburger menu**
3. Click **People** in the **Directory** section
4. For every user authorized to use AWS:
   1. Click the user's name
   2. Click **Profile**
   3. Click **Edit**
   4. In the _AWS\_RoleName_ input, enter the name (not the ARN) of the IAM role they should assume in your Substrate account (“Administrator” for yourself as you're getting started; if for others it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](../../mgmt/custom-iam-roles.html) first)
   5. Click **Save**

With your identity provider integrated, jump to [finishing up in your management account](../finishing.html).
