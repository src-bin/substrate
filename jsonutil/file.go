package jsonutil

import (
	"encoding/json"
	"os"
)

func Read(pathname string, document interface{}) error {
	b, err := os.ReadFile(pathname)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, document)
}

func Write(document interface{}, pathname string) error {
	b, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return err
	}
	b = append(b, '\n') // I wish there was a less wasteful way to do this
	return os.WriteFile(pathname, b, 0666)
}
