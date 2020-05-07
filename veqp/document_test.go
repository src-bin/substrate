package veqp

import "testing"

func TestValidate(t *testing.T) {
	d := &Document{
		ValidEnvironmentQualityPairs: []EnvironmentQualityPair{
			EnvironmentQualityPair{
				Environment: "development",
				Quality:     "alpha",
			},
			EnvironmentQualityPair{
				Environment: "development",
				Quality:     "beta",
			},
			EnvironmentQualityPair{
				Environment: "production",
				Quality:     "beta",
			},
			EnvironmentQualityPair{
				Environment: "production",
				Quality:     "gamma",
			},
		},
	}

	// Valid case.
	if err := d.Validate([]string{"development", "production"}, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Error(err)
	}

	// Every kind of invalid case.
	if err := d.Validate([]string{"development"}, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Log(err)
	} else {
		t.Error(err)
	}
	if err := d.Validate([]string{"development", "production"}, []string{"alpha", "beta"}); err != nil {
		t.Log(err)
	} else {
		t.Error(err)
	}
	if err := d.Validate([]string{"development", "staging", "production"}, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Log(err)
	} else {
		t.Error(err)
	}
	if err := d.Validate([]string{"development", "production"}, []string{"alpha", "beta", "gamma", "delta"}); err != nil {
		t.Log(err)
	} else {
		t.Error(err)
	}

}
