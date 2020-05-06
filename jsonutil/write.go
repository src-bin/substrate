package jsonutil

import (
	"encoding/json"
	"io/ioutil"
)

func Write(document interface{}, pathname string) error {
	b, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(pathname, b, 0666)
}
