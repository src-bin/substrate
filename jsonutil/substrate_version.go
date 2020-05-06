package jsonutil

import (
	"fmt"

	"github.com/src-bin/substrate/version"
)

type SubstrateVersion string

func (v SubstrateVersion) MarshalJSON() ([]byte, error) {
	if v == "" {
		v = SubstrateVersion(version.Version)
	}
	return []byte(fmt.Sprintf("%#v", v)), nil
}

func (SubstrateVersion) UnmarshalJSON([]byte) error { return nil }
