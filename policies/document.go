package policies

import (
	"encoding/json"
	"fmt"

	"github.com/src-bin/substrate/jsonutil"
)

type Document struct {
	Version   version
	Statement []Statement
}

func Unmarshal(s string) (*Document, error) {
	d := &Document{}
	return d, json.Unmarshal([]byte(s), d)
}

func (d *Document) Marshal() (string, error) {
	b, err := json.MarshalIndent(d, "", "\t")
	return string(b), err
}

type Statement struct {
	Effect    Effect
	Principal *Principal `json:",omitempty"`
	Action    jsonutil.StringSlice
	Resource  jsonutil.StringSlice `json:",omitempty"` // omitempty for AssumeRolePolicyDocument
	Condition Condition            `json:",omitempty"`
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
	AWS       jsonutil.StringSlice `json:",omitempty"`
	Federated jsonutil.StringSlice `json:",omitempty"`
	Service   jsonutil.StringSlice `json:",omitempty"`
}

func (p *Principal) String() string { return fmt.Sprintf("%+v", *p) }

type Condition map[string]map[string]string

type version struct{}

func (version) MarshalJSON() ([]byte, error) {
	return []byte(`"2012-10-17"`), nil
}

func (version) UnmarshalJSON([]byte) error { return nil }
