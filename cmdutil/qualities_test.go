package cmdutil

import (
	"testing"

	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/veqp"
)

func TestQualityForEnvironment0(t *testing.T) {
	doc := &veqp.Document{ValidEnvironmentQualityPairs: []veqp.EnvironmentQualityPair{}}
	quality := qualityForEnvironment("production", doc)
	if quality != "" {
		t.Fatalf("wrongly deduced quality %q from %s", quality, jsonutil.MustString(doc))
	}
}

func TestQualityForEnvironment1(t *testing.T) {
	doc := &veqp.Document{ValidEnvironmentQualityPairs: []veqp.EnvironmentQualityPair{
		{"production", "default"},
	}}
	quality := qualityForEnvironment("production", doc)
	if quality != "default" {
		t.Fatalf("wrongly deduced quality %q from %s", quality, jsonutil.MustString(doc))
	}
}

func TestQualityForEnvironment2(t *testing.T) {
	doc := &veqp.Document{ValidEnvironmentQualityPairs: []veqp.EnvironmentQualityPair{
		{"production", "beta"},
		{"production", "gamma"},
	}}
	quality := qualityForEnvironment("production", doc)
	if quality != "" {
		t.Fatalf("wrongly deduced quality %q from %s", quality, jsonutil.MustString(doc))
	}
}
