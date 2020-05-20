package oauthoidc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	ClientID      string
	baseURL       url.URL // not a pointer to force copying
	clientSecret  string
	pathQualifier func(string) string
}

func NewClient(
	host string,
	pathQualifier func(string) string,
	clientID, clientSecret string,
) *Client {
	return &Client{
		ClientID: clientID,
		baseURL: url.URL{
			Scheme: "https",
			Host:   host,
		},
		clientSecret:  clientSecret,
		pathQualifier: pathQualifier,
	}
}

// Get requests the given path with the given query string from the client's
// host and unmarshals the JSON response body into the given interface{}.  It
// returns the *http.Response, though its Body field is not usable, and an
// error, if any.
func (c *Client) Get(path string, query url.Values, i interface{}) (*http.Response, error) {
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
func (c *Client) Post(path string, body url.Values, i interface{}) (*http.Response, error) {
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

func (c *Client) URL(path string, query url.Values) *url.URL {
	u := c.baseURL // copy
	path = c.pathQualifier(path)
	if len(path) != 0 && path[0] != '/' {
		path = "/" + path
	}
	u.Path = path
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return &u
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
