package oauthoidc

import (
	"encoding/json"
	"net/url"
	"path"
)

const (
	OktaHostname                   = "OKTA_HOSTNAME"          // Lambda environment variable name
	OktaHostnameValueForGoogleIdP  = "unused-by-Google-IdP"   // old sentinel value
	OktaHostnameValueForNonOktaIdP = "unused-by-non-Okta-IdP" // new sentinel value
)

type OktaAccessToken struct {
	Audience string   `json:"aud"`
	ClientId string   `json:"cid"`
	DebugID  string   `json:"jti"`
	Expires  int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
	Issuer   string   `json:"iss"`
	Scopes   []string `json:"scp"`
	Subject  string   `json:"sub"`
	UserID   string   `json:"uid"`
	Version  int      `json:"ver"`
}

func (t *OktaAccessToken) JSONString() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *OktaAccessToken) Verify(c *Client) error {
	if t.ClientId != c.ClientId {
		return VerificationError{"cid", t.ClientId, c.ClientId}
	}
	if actual, expected := t.Issuer, c.URL(Issuer, nil).String(); actual != expected {
		return VerificationError{"iss", actual, expected}
	}
	return nil
}

func OktaPathQualifier(hostname string) PathQualifier {
	// TODO dynamically construct this function based on <https://${yourOktaDomain}/.well-known/openid-configuration>
	// per <https://developer.okta.com/docs/reference/api/oidc/>
	return func(p UnqualifiedPath) *url.URL {
		u := &url.URL{
			Scheme: "https",
			Host:   hostname,
		}
		switch p {
		case Authorize, Keys, Token:
			u.Path = path.Join("/oauth2", "v1", string(p))
		case Issuer:
			u.Path = ""
		case User:
			u.Path = "/api/v1/users/me"
		default:
			panic("unreachable")
		}
		return u
	}
}

func roleNameFromOktaIdP(c *Client, user string) (string, error) {
	var body struct {
		Profile struct {
			RoleName string `json:"AWS_RoleName"`
		} `json:"profile"`
		// lots of other fields that aren't relevant
	}
	_, _, err := c.Get(User, url.Values{}, &body)
	if err != nil {
		return "", err
	}
	//log.Printf("%+v", body)
	if body.Profile.RoleName != "" {
		return body.Profile.RoleName, nil
	}
	return "", UndefinedRoleError(user)
}
