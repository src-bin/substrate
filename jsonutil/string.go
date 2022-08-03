package jsonutil

import (
	"encoding/json"

	"github.com/src-bin/substrate/ui"
)

func MustOneLineString(document interface{}) string {
	s, err := OneLineString(document)
	if err != nil {
		ui.Fatal(err)
	}
	return s
}

func MustString(document interface{}) string {
	s, err := String(document)
	if err != nil {
		ui.Fatal(err)
	}
	return s
}

func OneLineString(document interface{}) (string, error) {
	b, err := json.Marshal(document)
	return string(b), err
}

func String(document interface{}) (string, error) {
	b, err := json.MarshalIndent(document, "", "\t")
	return string(b), err
}
