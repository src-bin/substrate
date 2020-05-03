package policies

import (
	"encoding/json"
	"fmt"
)

type Document struct {
	Version   version
	Statement []Statement
}

func (d *Document) JSON() (string, error) {
	b, err := json.MarshalIndent(d, "", "\t")
	return string(b), err
}

type Statement struct {
	Effect    Effect
	Principal *Principal `json:",omitempty"`
	Action    []string
	Resource  []string  `json:",omitempty"` // omitempty for AssumeRolePolicyDocument
	Condition Condition `json:",omitempty"`
}

type Effect string

const (
	Allow Effect = "Allow" // default, thanks to MarshalJSON
	Deny  Effect = "Deny"
)

func (e Effect) MarshalJSON() ([]byte, error) {
	switch e {
	case Allow, Deny:
	case "":
		e = Allow
	default:
		return nil, fmt.Errorf("invalid Effect %#v", e)
	}
	return []byte(fmt.Sprintf("%#v", e)), nil
}

type Principal struct {
	AWS       []string `json:",omitempty"`
	Federated []string `json:",omitempty"`
	Service   []string `json:",omitempty"`
}

type Condition map[string]map[string]string

type version struct{}

func (version) MarshalJSON() ([]byte, error) {
	return []byte(`"2012-10-17"`), nil
}
