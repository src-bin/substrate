package veqp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/version"
)

const Filename = "substrate.valid-environment-quality-pairs.json"

type Document struct {
	Admonition                   jsonutil.Admonition `json:"#"`
	ValidEnvironmentQualityPairs []EnvironmentQualityPair
	SubstrateVersion             jsonutil.SubstrateVersion
}

func ReadDocument() (*Document, error) {
	b, err := fileutil.ReadFile(Filename)
	if errors.Is(err, fs.ErrNotExist) {
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

func (d *Document) Ensure(environment, quality string) error {
	return d.EnsurePair(EnvironmentQualityPair{environment, quality})
}

func (d *Document) EnsurePair(eqp0 EnvironmentQualityPair) error {
	if d.ValidPair(eqp0) {
		return nil
	}
	d.ValidEnvironmentQualityPairs = append(d.ValidEnvironmentQualityPairs, eqp0)
	return d.Write()
}

func (d *Document) Len() int {
	return len(d.ValidEnvironmentQualityPairs)
}

// Valid returns true iff the given Environment and Quality appear together in
// the Document.
func (d *Document) Valid(environment, quality string) bool {
	return d.ValidPair(EnvironmentQualityPair{environment, quality})
}

// ValidPair returns true iff the given EnvironmentQualityPair appears in the
// Document.
func (d *Document) ValidPair(eqp0 EnvironmentQualityPair) bool {
	for _, eqp := range d.ValidEnvironmentQualityPairs {
		if eqp0 == eqp {
			return true
		}
	}
	return false
}

// Validate returns nil iff every environment and quality in the given slices
// appears in the Document and no environment or quality in the Document is
// missing from the given slices.
//
// Don't sort the arguments; their order is important.
func (d *Document) Validate(environments, qualities []string) error {
	for _, environment := range environments {
		if err := d.validateEnvironment(environment); err != nil {
			return err
		}
	}
	for _, quality := range qualities {
		if err := d.validateQuality(quality); err != nil {
			return err
		}
	}
	for _, eqp := range d.ValidEnvironmentQualityPairs {
		if err := d.validateEnvironmentQualityPair(eqp, environments, qualities); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) Write() error {
	return jsonutil.Write(d, Filename)
}

// validateEnvironment returns nil iff the given Environment appears in the Document.
func (d *Document) validateEnvironment(environment string) error {
	for _, eqp := range d.ValidEnvironmentQualityPairs {
		if environment == eqp.Environment {
			return nil
		}
	}
	return fmt.Errorf(`environment %q not paired with any quality`, environment)
}

// validateEnvironmentQualityPair returns nil iff both components of the given
// EnvironmentQualityPair appear in their respective given slices.
func (d *Document) validateEnvironmentQualityPair(
	eqp EnvironmentQualityPair,
	environments, qualities []string,
) error {
	validEnvironment := false
	for _, environment := range environments {
		if eqp.Environment == environment {
			validEnvironment = true
		}
	}
	if !validEnvironment {
		return fmt.Errorf(`environment %q is not valid`, eqp.Environment)
	}
	for _, quality := range qualities {
		if eqp.Quality == quality {
			return nil
		}
	}
	return fmt.Errorf(`quality %q is not valid`, eqp.Quality)
}

// validateQuality returns nil iff the given Quality appears in the Document.
func (d *Document) validateQuality(quality string) error {
	for _, eqp := range d.ValidEnvironmentQualityPairs {
		if quality == eqp.Quality {
			return nil
		}
	}
	return fmt.Errorf(`quality %q not paired with any environment`, quality)
}

type EnvironmentQualityPair struct {
	Environment, Quality string
}
