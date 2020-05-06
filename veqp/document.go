package veqp

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/version"
)

const Filename = "substrate.ValidEnvironmentQualityPairs.json"

type Document struct {
	Admonition                   jsonutil.Admonition `json:"#"`
	ValidEnvironmentQualityPairs []EnvironmentQualityPair
	SubstrateVersion             jsonutil.SubstrateVersion
}

func ReadDocument() (*Document, error) {
	b, err := fileutil.ReadFile(Filename)
	if errors.Is(err, os.ErrNotExist) {
		b = []byte("{}")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	d := &Document{}
	if err := json.Unmarshal(b, d); err != nil {
		return nil, err
	}

	// If d.SubstrateVersion != version.Version, migrate here.

	d.SubstrateVersion = jsonutil.SubstrateVersion(version.Version)
	return d, nil
}

func (d *Document) Ensure(eqp0 EnvironmentQualityPair) error {
	if d.Valid(eqp0) {
		return nil
	}
	d.ValidEnvironmentQualityPairs = append(d.ValidEnvironmentQualityPairs, eqp0)
	return d.Write()
}

func (d *Document) Environments() []string {
	return nil
}

func (d *Document) Qualities() []string {
	return nil
}

func (d *Document) Valid(eqp0 EnvironmentQualityPair) bool {
	for _, eqp := range d.ValidEnvironmentQualityPairs {
		if eqp0 == eqp {
			return true
		}
	}
	return false
}

func (d *Document) Write() error {
	return jsonutil.Write(d, Filename)
}

type EnvironmentQualityPair struct {
	Environment, Quality string
}
