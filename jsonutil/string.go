package jsonutil

import (
	"encoding/json"
	"log"
)

func MustString(document interface{}) string {
	s, err := String(document)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func String(document interface{}) (string, error) {
	b, err := json.MarshalIndent(document, "", "\t")
	return string(b), err
}
