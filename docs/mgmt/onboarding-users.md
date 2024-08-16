# Onboarding users

When new folks join your company they're probably going to need access to AWS. Here's a quick guide for granting it, depending on which identity provider you use.

After you've added folks to the identity provider per your usual onboarding process for all employees, do the following for each user who needs access to AWS.

## Azure AD

1. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_UsersAndTenants/UserManagementMenuBlade/\\\~/AllUsers](https://portal.azure.com/#view/Microsoft\_AAD\_UsersAndTenants/UserManagementMenuBlade/\\\~/AllUsers) in a browser (or visit the Azure portal, click **Azure Active Directory**, and click **Users**)
2. Click the user's name
3. Click **Assigned roles** in the left column
4. Click **Add assignments**
5. Select “Attribute Assignment Reader” and “Attribute Definition Reader”
6. Click **Add**
7. Click **Custom security attributes (preview)**
8. Click **Add assignment**
9. Select “AWS” in the _Attribute set_ column
10. Select “RoleName” in the _Attribute name_ column
11. Enter the name (not the ARN) of the IAM role they should assume in your Substrate account (“Administrator” for yourself as you're getting started; if for others it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](custom-iam-roles.html) first)
12. Click **Save**
13. Visit [https://portal.azure.com/#view/Microsoft\_AAD\_IAM/StartboardApplicationsMenuBlade/\~/AppAppsPreview/menuId\~/null](https://portal.azure.com/#view/Microsoft\_AAD\_IAM/StartboardApplicationsMenuBlade/\~/AppAppsPreview/menuId\~/null) in that same browser (or visit the Azure portal, click **Azure Active Directory**, and click **Enterprise applications**)
14. Click the name of the application you created above
15. Click **Users and groups** in the left column
16. Click **Add user/group**
17. Click **Users**
18. Select the user you're onboarding
19. Click **Select**
20. Click **Assign**

## Google Workspace

1. Visit [https://admin.google.com/ac/users](https://admin.google.com/ac/users) (or visit [https://admin.google.com](https://admin.google.com) and click **Users**)
2. Click the user's name
3. Click **User information**
4. In the _AWS_ section, click **Add RoleName** and paste the name (not the ARN) of the IAM role they should assume in your Substrate account (if it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](custom-iam-roles.html) first)
5. Click **SAVE**

## Okta

1. Visit your Okta admin panel in a browser
2. Click the **hamburger menu**
3. Click **People** in the **Directory** section
4. Click the user's name
5. Click **Profile**
6. Click **Edit**
7. In the _AWS\_RoleName_ input, enter the name (not the ARN) of the IAM role they should assume in your Substrate account (“Administrator” for yourself as you're getting started; if for others it's not “Administrator” or “Auditor”, ensure you've followed [adding non-Administrator roles for humans](custom-iam-roles.html) first)
8. Click **Save**
9. Click the **hamburger menu**
10. Click **Applications** in the **Applications** section
11. Click the name of your Intranet application
12. Click the **Assignments** tab
13. Click **Assign** and then **Assign to People**
14. Select your new folks
15. Click **Assign**
