package features

import (
	"os"
	"testing"
)

const testFeature feature = "testFeature"

func TestFeatureEnabled(t *testing.T) {
	testFeatureEnabled(t, testFeature, "", false)
	testFeatureEnabled(t, testFeature, "foo", false)
	testFeatureEnabled(t, testFeature, "foo,bar", false)
	testFeatureEnabled(t, testFeature, "testFeature", true)
	testFeatureEnabled(t, testFeature, "foo,testFeature", true)
	testFeatureEnabled(t, testFeature, "foo,testFeature,bar", true)
	testFeatureEnabled(t, testFeature, "testFeature,bar", true)
}

func testFeatureEnabled(t *testing.T, f feature, v string, expected bool) {
	os.Setenv(EnvironmentVariable, v)
	if actual := f.Enabled(); actual != expected {
		t.Errorf("%s.Enabled() with %s=%q got %v but expected %v", f, EnvironmentVariable, v, actual, expected)
	}
}
