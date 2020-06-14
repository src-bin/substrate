package oauthoidc

import (
	"encoding/json"
	"net/url"
	"path"
)

const OktaHostname = "OktaHostname"

type OktaAccessToken struct {
	Audience string   `json:"aud"`
	ClientID string   `json:"cid"`
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
	if t.ClientID != c.ClientID {
		return VerificationError{"cid", t.ClientID, c.ClientID}
	}
	if actual, expected := t.Issuer, c.URL(Issuer, nil).String(); actual != expected {
		return VerificationError{"iss", actual, expected}
	}
	return nil
}

func OktaPathQualifier(hostname, authServerId string) PathQualifier {
	// TODO dynamically construct this function based on <https://${yourOktaDomain}/.well-known/openid-configuration>
	// or <https://${yourOktaDomain}/oauth2/${authServerId}/.well-known/openid-configuration>
	// per <https://developer.okta.com/docs/reference/api/oidc/>
	return func(p UnqualifiedPath) *url.URL {
		u := &url.URL{
			Scheme: "https",
			Host:   hostname,
		}
		if p == Issuer {
			u.Path = path.Join("/oauth2", authServerId)
		} else {
			u.Path = path.Join("/oauth2", authServerId, "v1", string(p))
		}
		return u
	}
}
