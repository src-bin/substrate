package oauthoidc

import (
	"log"
	"net/url"
	"path"
)

func GoogleAdminDirectoryUser(c *Client, userKey string) ( /* TODO */ map[string]interface{}, error) {
	body := map[string]interface{}{} // TODO better type
	resp, err := c.GetURL(&url.URL{
		Scheme: "https",
		Host:   "admin.googleapis.com",
		Path:   path.Join("/admin/directory/v1/users", userKey),
	}, url.Values{"projection": {"full"}, "viewType": {"domain_public"}}, &body)
	if err != nil {
		return nil, err
	}
	log.Printf("resp: %+v", resp)
	log.Printf("body: %+v", body)
	return body, nil
}

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
