# Integrating your Azure AD identity provider

`substrate setup` will ask for several inputs, which this page will help you provide from your Azure AD identity provider.

These steps must be completed by an Azure administrator with the Application Administrator, Attribute Assignment Administrator, and Attribute Definition Administrator roles in an organization subscribed to Azure AD Premium 1 or Azure AD Premium 2.

## Create a custom security attribute for assigning IAM roles to Azure AD users

1. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_IAM/ActiveDirectoryMenuBlade/\~/CustomAttributesCatalog](https://portal.azure.com/#view/Microsoft\_AAD\_IAM/ActiveDirectoryMenuBlade/\~/CustomAttributesCatalog) in a browser (or visit the Azure portal, click **Azure Active Directory**, and click **Custom security attributes (Preview)**)
2. Click **Add attribute set**
3. Enter “AWS” for _Attribute set name_
4. Click **Add**
5. Click **AWS**
6. Click **Add attribute**
7. Enter “RoleName” for _Attribute name_
8. Click **Save**

## Create and configure an OAuth OIDC client

1. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_IAM/ActiveDirectoryMenuBlade/\~/RegisteredApps](https://portal.azure.com/#view/Microsoft\_AAD\_IAM/ActiveDirectoryMenuBlade/\~/RegisteredApps) in a browser (or visit the Azure portal, click **Azure Active Directory**, and click **App registrations**)
2. Click **New registration**
3. Enter a name for the application
4. Select “Accounts in this organizational directory only (Default Directory only - Single tenant)”
5. Select “Web” as the platform
6. Enter “https://_intranet-dns-domain-name_/login” (substituting your just-purchased or just-transferred Intranet DNS domain name) next to the platform selector in the text input with _e.g. https://example.com/auth_ as its placeholder
7. Click **Register**
8. Use the _Application (client) ID_ to respond to `substrate setup`'s prompt
9. Click **Add a certificate or secret**
10. Click **New client secret**
11. Enter a description
12. Specify an expiration date however you see fit and set yourself a reminder to rotate the client secret before that date arrives
13. Click **Add**
14. Use the _Value_ to respond to `substrate setup`'s prompt (being wary that this value will never be shown again; if you need to copy it again, you'll need to create a new client secret)

## Authorize users to use AWS

1. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_UsersAndTenants/UserManagementMenuBlade/\\\~/AllUsers](https://portal.azure.com/#view/Microsoft\_AAD\_UsersAndTenants/UserManagementMenuBlade/\\\~/AllUsers) in a browser (or visit the Azure portal, click **Azure Active Directory**, and click **Users**)
2. For every user authorized to use AWS:
   1. Click the user's name
   2. Click **Assigned roles** in the left column
   3. Click **Add assignments**
   4. Select “Attribute Assignment Reader” and “Attribute Definition Reader”
   5. Click **Add**
   6. Click **Custom security attributes (preview)** in the left column
   7. Click **Add assignment**
   8. Select “AWS” in the _Attribute set_ column
   9. Select “RoleName” in the _Attribute name_ column
   10. Enter the name (not the ARN) of the IAM role they should assume in your Substrate account (“Administrator” for yourself as you're getting started; if for others it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](../../mgmt/custom-iam-roles.md) first)
   11. Click **Save**
3. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_IAM/StartboardApplicationsMenuBlade/\~/AppAppsPreview/menuId\~/null](https://portal.azure.com/#view/Microsoft\_AAD\_IAM/StartboardApplicationsMenuBlade/\~/AppAppsPreview/menuId\~/null) in that same browser (or visit the Azure portal, click **Azure Active Directory**, and click **Enterprise applications**)
4. Click the name of the application you created above
5. Click **Users and groups** in the left column
6. Click **Add user/group**
7. Click **Users**
8. Select every user authorized to use AWS
9. Click **Select**
10. Click **Assign**

With your identity provider integrated, jump to [finishing up in your management account](../finishing.md).
