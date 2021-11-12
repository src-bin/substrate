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

type Client struct {
	ClientID      string
	clientSecret  string
	pathQualifier PathQualifier
}

func NewClient(
	sess *session.Session,
	stageVariables map[string]string,
) (*Client, error) {
	clientSecret, err := awssecretsmanager.CachedSecret(
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

	var pathQualifier PathQualifier
	if hostname := stageVariables[OktaHostname]; hostname == OktaHostnameValueForGoogleIDP {
		pathQualifier = GooglePathQualifier()
	} else {
		pathQualifier = OktaPathQualifier(hostname, "default")
	}
	return &Client{
		ClientID:      stageVariables[OAuthOIDCClientID],
		clientSecret:  clientSecret,
		pathQualifier: pathQualifier,
	}, nil
}

// Get requests the given path with the given query string from the client's
// host and unmarshals the JSON response body into the given interface{}.  It
// returns the *http.Response, though its Body field is not usable, and an
// error, if any.
func (c *Client) Get(path UnqualifiedPath, query url.Values, i interface{}) (*http.Response, error) {
	u := c.URL(path, query)
	resp, err := http.DefaultClient.Do(c.request("GET", u))
	if err != nil {
		return resp, err
	}
	return resp, unmarshalJSON(resp, i)
}

// Post requests the given path with the given body (form-encoded) from the
// client's host and unmarshals the JSON response body into the given
// interface{}.  It returns the *http.Response, though its Body field is not
// usable, and an error, if any.
func (c *Client) Post(path UnqualifiedPath, body url.Values, i interface{}) (*http.Response, error) {
	u := c.URL(path, nil)
	req := c.request("POST", u)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = ioutil.NopCloser(strings.NewReader(body.Encode()))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, unmarshalJSON(resp, i)
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
	if c.ClientID != "" && c.clientSecret != "" {
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

type PathQualifier func(UnqualifiedPath) *url.URL

type UnqualifiedPath string

// Names of well-known URLs in the OAuth OIDC flow.  PathQualifiers will turn
// these into the actual fully-qualified URLs used by supported IdPs.
const (
	Authorize UnqualifiedPath = "authorize"
	Issuer    UnqualifiedPath = "issuer"
	Keys      UnqualifiedPath = "keys"
	Token     UnqualifiedPath = "token"
)
