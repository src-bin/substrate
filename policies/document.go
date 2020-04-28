package policies

import "fmt"

type Document struct {
	Version   version
	Statement []Statement
}

type Statement struct {
	Effect    Effect
	Principal Principal
	Action    []string
	Resource  []string
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
	AWS     []string `json:",omitempty"`
	Service []string `json:",omitempty"`
}

type Condition map[string]map[string]string

type version struct{}

func (version) MarshalJSON() ([]byte, error) {
	return []byte(`"2012-10-17"`), nil
}
