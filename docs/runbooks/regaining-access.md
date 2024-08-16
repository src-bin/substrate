# Regaining access in case the Credential and Instance Factories are broken

If for some reason your Credential and Instance Factories are both broken, which could in fact be because the API Gateway authenticator and/or authorizer that ties them to your identity provider are broken, you'll find yourself feeling pretty locked out.

If you happen to have credentials in your shell environment that are still valid, you can run `substrate setup` to remedy the situation.

More likely, though, you'll need to follow these steps:

1. Login to the AWS Console using the root email address, password, and second factor for your organization's management account
2. Visit [https://console.aws.amazon.com/iam/home#/security\_credentials](https://console.aws.amazon.com/iam/home#/security\_credentials)
3. Open the **Access keys (access key ID and secret access key)** section
4. Click **Create New Access Key**
5. Provide the resulting access key ID and secret access key to `substrate setup`
6. When your Credential and Instance Factories return to service, delete the access key you created in step 4
