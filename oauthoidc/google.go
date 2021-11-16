package oauthoidc

import (
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/src-bin/substrate/roles"
)

func GooglePathQualifier() PathQualifier {
	// TODO dynamically construct this function from <https://accounts.google.com/.well-known/openid-configuration>
	return func(p UnqualifiedPath) *url.URL {
		switch p {
		case Authorize:
			return &url.URL{
				Scheme: "https",
				Host:   "accounts.google.com",
				Path:   "/o/oauth2/v2/auth",
			}
		case Issuer:
			return &url.URL{
				Scheme: "https",
				Host:   "accounts.google.com",
			}
		case Keys:
			return &url.URL{
				Scheme: "https",
				Host:   "www.googleapis.com",
				Path:   "/oauth2/v3/certs",
			}
		case Token:
			return &url.URL{
				Scheme: "https",
				Host:   "oauth2.googleapis.com",
				Path:   "/token",
			}
		}
		panic("unreachable")
	}
}

func RoleNameFromGoogleIdP(c *Client, user string) (string, error) {
	var body struct {
		CustomSchemas map[string]map[string]string `json:"customSchemas"`
		PrimaryEmail  string                       `json:"primaryEmail"`
		// lots of other fields that aren't relevant
	}
	resp, err := c.GetURL(&url.URL{
		Scheme: "https",
		Host:   "admin.googleapis.com",
		Path:   path.Join("/admin/directory/v1/users", user),
	}, url.Values{"projection": {"full"}, "viewType": {"domain_public"}}, &body)
	if err != nil {
		return "", err
	}
	log.Printf("resp: %+v", resp)
	log.Printf("body: %+v", body)
	const awsCategory = "AWS" // there's a risk the value we want is under "AWS1234" (or some such) since Google papers over duplicate category names in the UI
	for category, m := range body.CustomSchemas {
		if category == awsCategory {
			for name, value := range m {
				if name == "RoleName" {
					return value, nil
				}
			}
		}
	}

	// Also check for (and then parse) the original AWS.Role attribute that
	// included a role and SAML provider ARN with a comma between them.
	for category, m := range body.CustomSchemas {
		if category == awsCategory {
			for name, value := range m {
				if name == "Role" {
					return roles.Name(strings.Split(value, ",")[0])
				}
			}
		}
	}

	return "", UndefinedRoleError(user)
}
