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

func ParseAndVerifyJWT(s string, c *Client, p JWTPayload) (*JWT, error) {
	jwt, err := ParseJWT(s, p)
	if err != nil {
		return nil, err
	}
	if err := jwt.Verify(c); err != nil {
		return nil, err
	}
	return jwt, nil
}

func ParseJWT(s string, p JWTPayload) (*JWT, error) {
	slice := strings.Split(s, ".")

	jwt := &JWT{}
	var err error
	jwt.Header, err = parseJWTHeader(slice[0])
	if err != nil {
		return nil, err
	}
	jwt.Payload, err = parseJWTPayload(slice[1], p)
	if err != nil {
		return nil, err
	}
	if len(slice) != 3 {
		return nil, MalformedJWTError(fmt.Sprintf(
			"JWTs should have 3 parts but %d found (so far: %+v)",
			len(slice),
			jwt,
		))
	}
	jwt.Signature, err = parseJWTSignature(slice[2])
	if err != nil {
		return nil, err
	}
	jwt.signingInput = []byte(slice[0] + "." + slice[1])
	return jwt, nil
}

func (jwt *JWT) Verify(c *Client) error {

	key, err := jwt.findKey(c)
	if err != nil {
		return err
	}

	pub, err := key.RSAPublicKey()
	if err != nil {
		return err
	}
	hashed := sha256.Sum256(jwt.signingInput)
	if err := rsa.VerifyPKCS1v15(
		pub,
		crypto.SHA256,
		hashed[:],
		[]byte(jwt.Signature),
	); err != nil {
		return err
	}

	if err := jwt.Payload.Verify(c); err != nil {
		return err
	}

	return nil
}

func (jwt *JWT) findKey(c *Client) (*Key, error) {
	// TODO memoize doc.Keys
	doc := &KeysResponse{}
	if _, _, err := c.Get(Keys, nil, doc); err != nil {
		return nil, err
	}
	for _, key := range doc.Keys {
		if jwt.Header.KeyID == key.KeyID {
			return key, nil
		}
	}
	return nil, KeyNotFoundError(jwt.Header.KeyID)
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

type JWTPayload interface {
	Verify(c *Client) error
}

func parseJWTPayload(s string, p JWTPayload) (JWTPayload, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, p); err != nil {
		return nil, err
	}

	return p, nil
}

type JWTSignature []byte

func parseJWTSignature(s string) (JWTSignature, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return JWTSignature(b), nil
}

type InvalidJWTError string

func (err InvalidJWTError) Error() string {
	return fmt.Sprintf("InvalidJWTError: %s", string(err))
}

type KeyNotFoundError string

func (err KeyNotFoundError) Error() string {
	return fmt.Sprintf("KeyNotFoundError: %s not found", string(err))
}

type MalformedJWTError string

func (err MalformedJWTError) Error() string {
	return fmt.Sprintf("MalformedJWTError: %s", string(err))
}

type VerificationError struct {
	Field            string
	Actual, Expected string
}

func (err VerificationError) Error() string {
	return fmt.Sprintf("VerificationError: expected %q to be %q but got %q", err.Field, err.Expected, err.Actual)
}
