package oauthoidc

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
)

func Nonce() (string, error) {
	buf := &bytes.Buffer{}
	b := base64.NewEncoder(base64.URLEncoding, buf)
	_, err := io.CopyN(b, rand.Reader, 12)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
