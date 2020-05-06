package jsonutil

type Admonition struct{}

func (Admonition) MarshalJSON() ([]byte, error) {
	return []byte(`"managed by Substrate and synchronized with AWS via Terraform; do not edit by hand"`), nil
}

func (Admonition) UnmarshalJSON([]byte) error { return nil }
