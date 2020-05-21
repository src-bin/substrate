package oauthoidc

import (
	"encoding/base64"
	"encoding/json"
)

type State struct {
	Next  string
	Nonce string
}

func ParseState(s string) (*State, error) {
	if s == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	state := &State{}
	if err := json.Unmarshal(b, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err) // this feels wrong but there's just no way it goes wrong
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
