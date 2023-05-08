package policies

import (
	"encoding/json"
	"fmt"

	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/ui"
)

type Document struct {
	Version   version
	Statement []Statement // annoyingly signular because AWS made it singular
}

func AssumeRolePolicyDocument(principal *Principal) *Document {
	doc := &Document{
		Statement: []Statement{{
			Principal: principal,
			Action:    []string{"sts:AssumeRole"},
		}},
	}

	// Infer from the type of principal whether we additionally need a condition on this statement per
	// <https://help.okta.com/en/prod/Content/Topics/DeploymentGuides/AWS/connect-okta-single-aws.htm>.
	if principal.Federated != nil {
		for i := 0; i < len(doc.Statement); i++ {
			doc.Statement[i].Action[0] = "sts:AssumeRoleWithSAML"
			doc.Statement[i].Condition = Condition{"StringEquals": {"SAML:aud": []string{"https://signin.aws.amazon.com/saml"}}}
		}
	}

	return doc
}

func Merge(docs ...*Document) *Document {
	doc := &Document{}
	for i := 0; i < len(docs); i++ {
		doc.Statement = append(doc.Statement, docs[i].Statement...)
	}
	return doc
}

func Unmarshal(b []byte) (*Document, error) {
	d := &Document{}
	return d, json.Unmarshal(b, d)
}

func UnmarshalString(s string) (*Document, error) {
	return Unmarshal([]byte(s))
}

func (d *Document) Marshal() (string, error) {
	b, err := json.MarshalIndent(d, "", "\t")
	return string(b), err
}

func (d *Document) MustMarshal() string {
	s, err := d.Marshal()
	if err != nil {
		ui.Fatal(err)
	}
	return s
}

type Statement struct {
	Effect    Effect
	Principal *Principal `json:",omitempty"`
	Action    jsonutil.StringSlice
	Resource  jsonutil.StringSlice `json:",omitempty"` // omitempty for AssumeRolePolicyDocument
	Condition Condition            `json:",omitempty"`
	Sid       string               `json:",omitempty"`
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

func (e Effect) String() string {
	return string(e)
}

type Principal struct {
	AWS       jsonutil.StringSlice `json:",omitempty"`
	Federated jsonutil.StringSlice `json:",omitempty"`
	Service   jsonutil.StringSlice `json:",omitempty"`
}

func (p *Principal) String() string { return fmt.Sprintf("%+v", *p) }

type Condition map[string]map[string]jsonutil.StringSlice

type version struct{}

func (version) MarshalJSON() ([]byte, error) {
	return []byte(`"2012-10-17"`), nil
}

func (version) UnmarshalJSON([]byte) error { return nil }
