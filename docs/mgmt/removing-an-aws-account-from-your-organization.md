# Removing an AWS account from your organization

From time to time you may have a reason to remove an AWS account from your organization. The process is unfortunately tedious.

Before you begin this process, note well that AWS enforces a waiting period of “a few days” (which they really do neglect to specify more precisely) between an account joining an organization, whether by invitation or creation, and that account leaving the organization. If you're met with this error (or you know you will be), do what it says and try again in a few days.

1. Visit [https://console.aws.amazon.com](https://console.aws.amazon.com) in an incognito window
2. Leave “Root user” selected
3. Enter the email address of the account (which you can find in `substrate.accounts.txt` or by the rules below)
   * If you're closing an account that `substrate account create` created, the email address is the same as you used for your management account with “+_domain_-_environment_-_quality_” appended to the local part
   * If you're closing an account you invited into your organization or created manually and then used `substrate account adopt`, the email address is unchanged from what it was
4. Click **Next**
5. Click **Forgot password?**
6. Respond to the captcha and click **Send email**
7. Open the link emailed to you in an incognito window
8. Reset the password and, after that's finished, click **Sign in**
9. Sign in using that same email address and the password you just set
10. Visit [https://console.aws.amazon.com/organizations/home?#/organization/overview](https://console.aws.amazon.com/organizations/home?#/organization/overview)
11. Click **Leave organization**
12. Confirm; click **Leave organization** again
13. Click **Complete the account sign-up steps**
14. Provide payment information
15. Verify your phone number
16. Select the free support plan
17. When returned to the AWS Organizations console, click **Leave organization** again and confirm

## References

* [https://docs.aws.amazon.com/organizations/latest/userguide/orgs\_manage\_accounts\_close.html](https://docs.aws.amazon.com/organizations/latest/userguide/orgs\_manage\_accounts\_close.html)
* [https://docs.aws.amazon.com/organizations/latest/userguide/orgs\_manage\_accounts\_access.html#orgs\_manage\_accounts\_access-as-root](https://docs.aws.amazon.com/organizations/latest/userguide/orgs\_manage\_accounts\_access.html#orgs\_manage\_accounts\_access-as-root)
