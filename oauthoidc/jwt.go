package oauthoidc

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type JWT struct {
	Header       *JWTHeader
	Payload      JWTPayload
	Signature    JWTSignature
	signingInput []byte
}

func ParseAndVerifyJWT(s string, c *Client, v interface{}) (*JWT, error) {
	jwt, err := ParseJWT(s, v)
	if err != nil {
		return nil, err
	}
	if err := jwt.Verify(c); err != nil {
		return nil, err
	}
	return jwt, nil
}

func ParseJWT(s string, v interface{}) (*JWT, error) {
	slice := strings.Split(s, ".")
	if len(slice) != 3 {
		return nil, MalformedJWTError(fmt.Sprintf(
			"JWTs should have 3 parts but %d found",
			len(slice),
		))
	}

	jwt := &JWT{}
	var err error
	jwt.Header, err = parseJWTHeader(slice[0])
	if err != nil {
		return nil, err
	}
	jwt.Payload, err = parseJWTPayload(slice[1], v)
	if err != nil {
		return nil, err
	}
	jwt.Signature, err = parseJWTSignature(slice[2])
	if err != nil {
		return nil, err
	}
	jwt.signingInput = []byte(slice[0] + "." + slice[1])
	return jwt, nil
}

func (jwt *JWT) Verify(c *Client) error {
	doc := &OktaKeysResponse{}
	if _, err := c.Get(KeysPath, nil, doc); err != nil {
		return err
	}
	for _, key := range doc.Keys {
		if jwt.Header.KeyID == key.KeyID {

			pub, err := key.RSAPublicKey()
			if err != nil {
				return err
			}
			hashed := sha256.Sum256(jwt.signingInput)
			return rsa.VerifyPKCS1v15(
				pub,
				crypto.SHA256,
				hashed[:],
				[]byte(jwt.Signature),
			)

		}
	}
	return KeyNotFoundError(jwt.Header.KeyID)
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"` // maybe Okta-specific
}

func parseJWTHeader(s string) (*JWTHeader, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	h := &JWTHeader{}
	if err := json.Unmarshal(b, h); err != nil {
		return nil, err
	}
	return h, nil
}

type JWTPayload interface{}

func parseJWTPayload(s string, v interface{}) (JWTPayload, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return nil, err
	}
	return v, nil
}

type JWTSignature []byte

func parseJWTSignature(s string) (JWTSignature, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return JWTSignature(b), nil
}

type KeyNotFoundError string

func (err KeyNotFoundError) Error() string {
	return fmt.Sprintf("KeyNotFoundError: %v not found", string(err))
}

type MalformedJWTError string

func (err MalformedJWTError) Error() string {
	return fmt.Sprintf("MalformedJWTError: %v", string(err))
}
