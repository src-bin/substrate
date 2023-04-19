package oauthoidc

import (
	"net/url"
	"path"
)

const (
	AzureADTenantId                    = "AZURE_AD_TENANT_ID"        // Lambda environment variable name
	AzureADTenantValueForNonAzureADIdP = "unused-by-non-AzureAD-IdP" // sentinel value
)

func AzureADPathQualifier(tenantId string) PathQualifier {
	// TODO dynamically construct this function based on <https://login.microsoftonline.com/${tenantId}/v2.0/.well-known/openid-configuration>
	return func(p UnqualifiedPath) *url.URL {
		u := &url.URL{
			Scheme: "https",
			Host:   "login.microsoftonline.com",
			Path:   path.Join("/", tenantId),
		}
		switch p {
		case Authorize:
			u.Path = path.Join(u.Path, "oauth2/v2.0/authorize")
		case Issuer:
			u.Path = path.Join(u.Path, "v2.0")
		case Keys:
			u.Path = path.Join(u.Path, "discovery/v2.0/keys")
		case Token:
			u.Path = path.Join(u.Path, "oauth2/v2.0/token")
		default:
			panic("unreachable")
		}
		//log.Printf("Azure AD URL for %s: %s", p, u)
		return u
	}
}

func roleNameFromAzureADIdP(c *Client, user string) (string, error) {
	var body struct {
		CustomSecurityAttributes struct {
			AWS struct {
				RoleName string
			}
		} `json:"customSecurityAttributes"`
	}
	_, _, err := c.GetURL(&url.URL{
		Scheme: "https",
		Host:   "graph.microsoft.com",
		Path:   "/beta/me",
	}, url.Values{
		"$select": {"customSecurityAttributes"},
	}, &body)
	if err != nil {
		return "", err
	}
	//log.Printf("body: %+v", body)
	if body.CustomSecurityAttributes.AWS.RoleName != "" {
		return body.CustomSecurityAttributes.AWS.RoleName, nil
	}

	return "", UndefinedRoleError(user)
}
