package oauthoidc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
)

const (
	OAuthOIDCClientID              = "OAuthOIDCClientID"
	OAuthOIDCClientSecret          = "OAuthOIDCClientSecret"
	OAuthOIDCClientSecretTimestamp = "OAuthOIDCClientSecretTimestamp"
)

const (
	ProviderGoogle Provider = iota
	ProviderOkta
)

type Client struct {
	AccessToken   string
	ClientID      string
	clientSecret  string
	pathQualifier PathQualifier
	provider      Provider
}

func NewClient(
	sess *session.Session,
	stageVariables map[string]string,
) (*Client, error) {
	clientSecret, err := awssecretsmanager.CachedSecret( // TODO memoize
		secretsmanager.New(sess),
		fmt.Sprintf(
			"%s-%s",
			OAuthOIDCClientSecret,
			stageVariables[OAuthOIDCClientID],
		),
		stageVariables[OAuthOIDCClientSecretTimestamp],
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		ClientID:     stageVariables[OAuthOIDCClientID],
		clientSecret: clientSecret,
	}
	if hostname := stageVariables[OktaHostname]; hostname == OktaHostnameValueForGoogleIdP /* begin remove in 2021.12 */ || hostname == "unused-by-Google-IDP" /* end remove in 2021.12 */ {
		c.pathQualifier = GooglePathQualifier()
		c.provider = ProviderGoogle
	} else {
		c.pathQualifier = OktaPathQualifier(hostname, "default")
		c.provider = ProviderOkta
	}
	return c, nil
}

// Get requests the given path with the given query string from the client's
// host and unmarshals the JSON response body into the given interface{}.  It
// returns the *http.Response, though its Body field is not usable, and an
// error, if any.
func (c *Client) Get(path UnqualifiedPath, query url.Values, i interface{}) (*http.Response, error) {
	return c.GetURL(c.URL(path, query), nil, i)
}

func (c *Client) GetURL(u *url.URL, query url.Values, i interface{}) (*http.Response, error) {
	if query != nil {
		u.RawQuery = query.Encode()
	}
	resp, err := http.DefaultClient.Do(c.request("GET", u))
	if err != nil {
		return resp, err
	}
	return resp, unmarshalJSON(resp, i)
}

func (c *Client) IsGoogle() bool { return c.provider == ProviderGoogle }

func (c *Client) IsOkta() bool { return c.provider == ProviderOkta }

// Post requests the given path with the given body (form-encoded) from the
// client's host and unmarshals the JSON response body into the given
// interface{}.  It returns the *http.Response, though its Body field is not
// usable, and an error, if any.
func (c *Client) Post(path UnqualifiedPath, body url.Values, i interface{}) (*http.Response, error) {
	return c.PostURL(c.URL(path, nil), body, i)
}

func (c *Client) PostURL(u *url.URL, body url.Values, i interface{}) (*http.Response, error) {
	req := c.request("POST", u)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = ioutil.NopCloser(strings.NewReader(body.Encode()))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, unmarshalJSON(resp, i)
}

func (c *Client) Provider() Provider { return c.provider }

func (c *Client) RoleNameFromIdP(user string) (string, error) {
	switch c.provider {
	case ProviderGoogle:
		return roleNameFromGoogleIdP(c, user)
	case ProviderOkta:
		return roleNameFromOktaIdP()
	}
	return "", UndefinedRoleError(fmt.Sprintf("%s IdP", c.provider))
}

func (c *Client) URL(path UnqualifiedPath, query url.Values) *url.URL {
	u := c.pathQualifier(path)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u
}

func (c *Client) request(method string, u *url.URL) *http.Request {
	req := &http.Request{
		Body:       nil,
		Header:     make(http.Header),
		Host:       u.Host,
		Method:     method,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        u,
	}
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	} else if c.ClientID != "" && c.clientSecret != "" {
		req.Header.Set(
			"Authorization",
			"Basic "+base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
				"%s:%s",
				c.ClientID,
				c.clientSecret,
			))),
		)
	}
	return req
}

type PathQualifier func(UnqualifiedPath) *url.URL

type Provider int // do not persist this type, as its values are assigned by iota and aren't stable

type UnqualifiedPath string

// Names of well-known URLs in the OAuth OIDC flow.  PathQualifiers will turn
// these into the actual fully-qualified URLs used by supported IdPs.
const (
	Authorize UnqualifiedPath = "authorize"
	Issuer    UnqualifiedPath = "issuer"
	Keys      UnqualifiedPath = "keys"
	Token     UnqualifiedPath = "token"
)

func unmarshalJSON(resp *http.Response, i interface{}) error {
	if i == nil {
		return nil
	}
	defer resp.Body.Close()
	/*
		if err := json.NewDecoder(resp.Body).Decode(doc); err != nil {
			return err
		}
	*/
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, i); err != nil {
		return err
	}
	return nil
}
