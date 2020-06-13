package oauthoidc

import "net/url"

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
