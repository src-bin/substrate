package jsonutil

type Admonition struct{}

func (Admonition) MarshalJSON() ([]byte, error) {
	return []byte(`"managed by Substrate; do not edit by hand"`), nil
}

func (Admonition) UnmarshalJSON([]byte) error { return nil }
