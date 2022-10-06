package oauthoidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awssecretsmanager"
)

const (
	OAuthOIDCClientID              = "OAUTH_OIDC_CLIENT_ID"               // Lambda environment variable name
	OAuthOIDCClientSecret          = "OAuthOIDCClientSecret"              // Secrets Manager secret name
	OAuthOIDCClientSecretTimestamp = "OAUTH_OIDC_CLIENT_SECRET_TIMESTAMP" // Lambda environment variable name
)

type Client struct {
	AccessToken   string
	ClientID      string
	clientSecret  string
	memoizedKeys  []*Key
	pathQualifier PathQualifier
	provider      Provider
}

func NewClient(
	ctx context.Context,
	cfg *awscfg.Config,
	clientID string,
	clientSecretTimestamp string, // for finding the real client secret in Secrets Manager
	hostname string,
) (*Client, error) {
	c := &Client{ClientID: clientID}
	if hostname == OktaHostnameValueForGoogleIdP {
		c.pathQualifier = GooglePathQualifier()
		c.provider = Google
	} else {
		c.pathQualifier = OktaPathQualifier(hostname, "default")
		c.provider = Okta
	}

	// Getch the client secret from AWS Secrets Manager.
	clientSecret, err := awssecretsmanager.CachedSecret(
		ctx,
		cfg,
		fmt.Sprintf(
			"%s-%s",
			OAuthOIDCClientSecret,
			clientID,
		),
		clientSecretTimestamp,
	)
	if err != nil {
		return nil, err
	}
	c.clientSecret = clientSecret

	// Prefetch the public keys for verifying JWT signatures.
	if _, err := c.Keys(); err != nil {
		return nil, err
	}

	return c, nil
}

// Get requests the given path with the given query string from the client's
// host and unmarshals the JSON response body into the given interface{}.  It
// returns the *http.Response, though its Body field is not usable, and an
// error, if any.
func (c *Client) Get(path UnqualifiedPath, query url.Values, i interface{}) (*http.Response, []byte, error) {
	return c.GetURL(c.URL(path, query), nil, i)
}

func (c *Client) GetURL(u *url.URL, query url.Values, i interface{}) (*http.Response, []byte, error) {
	if query != nil {
		u.RawQuery = query.Encode()
	}
	resp, err := http.DefaultClient.Do(c.request("GET", u))
	if err != nil {
		return resp, nil, err
	}
	return unmarshalJSON(resp, i)
}

func (c *Client) IsGoogle() bool { return c.provider == Google }

func (c *Client) IsOkta() bool { return c.provider == Okta }

// Keys returns the OAuth OIDC provider's current list of public keys,
// memoizing the response for the rest of this process's lifetime.
// Google's Cache-Control header suggests they rotate keys every few hours;
// Okta's suggests they rotate keys about every two months. This client is
// 's expected to run in Lambda so it seems safe to not invalidate.
func (c *Client) Keys() ([]*Key, error) {
	defer func(t0 time.Time) { ui.PrintWithCaller(time.Since(t0)) }(time.Now())
	if c.memoizedKeys != nil {
		return c.memoizedKeys, nil
	}
	doc := &KeysResponse{}
	if _, _, err := c.Get(Keys, nil, doc); err != nil {
		return nil, err
	}
	c.memoizedKeys = doc.Keys
	return c.memoizedKeys, nil
}

// Post requests the given path with the given body (form-encoded) from the
// client's host and unmarshals the JSON response body into the given
// interface{}.  It returns the *http.Response, though its Body field is not
// usable, and an error, if any.
func (c *Client) Post(path UnqualifiedPath, body url.Values, i interface{}) (*http.Response, []byte, error) {
	return c.PostURL(c.URL(path, nil), body, i)
}

func (c *Client) PostURL(u *url.URL, body url.Values, i interface{}) (*http.Response, []byte, error) {
	req := c.request("POST", u)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = ioutil.NopCloser(strings.NewReader(body.Encode()))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, nil, err
	}
	return unmarshalJSON(resp, i)
}

func (c *Client) Provider() Provider { return c.provider }

func (c *Client) RoleNameFromIdP(user string) (string, error) {
	switch c.provider {
	case Google:
		return roleNameFromGoogleIdP(c, user)
	case Okta:
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

type Provider string

const (
	Google Provider = "Google"
	Okta   Provider = "Okta"
)

type UnqualifiedPath string

// Names of well-known URLs in the OAuth OIDC flow.  PathQualifiers will turn
// these into the actual fully-qualified URLs used by supported IdPs.
const (
	Authorize UnqualifiedPath = "authorize"
	Issuer    UnqualifiedPath = "issuer"
	Keys      UnqualifiedPath = "keys"
	Token     UnqualifiedPath = "token"
)

func unmarshalJSON(
	resp *http.Response,
	i interface{},
) (
	*http.Response, // pass-through
	[]byte, // unparsed body
	error, // I/O or JSON error
) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, body, err
	}
	if i != nil {
		err = json.Unmarshal(body, i)
	}
	return resp, body, err
}
