package oauthoidc

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"path"
)

const (
	AuthorizePath = "authorize"
	KeysPath      = "keys"
	TokenPath     = "token"
)

func OktaPathQualifier(basePath string) func(string) string {
	return func(p string) string {
		return path.Join(basePath, p)
	}
}

type OktaKey struct {
	Algorithm string `json:"alg"`
	Exponent  string `json:"e"` // comes base64-URL-encoded
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Modulus   string `json:"n"` // comes base64-URL-encoded
	Status    string `json:"status"`
	Use       string `json:"use"`
}

func (k *OktaKey) RSAPublicKey() (*rsa.PublicKey, error) {

	e, err := base64.RawURLEncoding.DecodeString(k.Exponent)
	if err != nil {
		return nil, err
	}
	e4 := make([]byte, 4)
	copy(e4[len(e4)-len(e):], e) // it may come off the wire in too-compact a representation

	n, err := base64.RawURLEncoding.DecodeString(k.Modulus)
	if err != nil {
		return nil, err
	}

	i := &big.Int{}
	return &rsa.PublicKey{
		E: int(binary.BigEndian.Uint32(e4)),
		N: i.SetBytes(n),
	}, nil
}

type OktaKeysResponse struct {
	Keys []*OktaKey `json:"keys"`
}

type OktaTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	IDToken     string `json:"id_token"`
}
