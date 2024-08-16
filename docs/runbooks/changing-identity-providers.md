# Changing identity providers

Suppose when you began using Substrate you chose to use Google as your identity provider but now you've grown and decided to make the leap to Azure AD or Okta. Here's how to proceed:

1. `rm -f substrate.azure-ad-tenant substrate.oauth-oidc-client-id substrate.oauth-oidc-client-secret-timestamp substrate.okta-hostname`
2. Follow the [integrating your identity provider to control access to AWS](../bootstrapping/integrating-your-identity-provider/) section of the getting started guide again
