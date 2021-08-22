package ui

import (
	"encoding/json"
	"os"
)

func PrettyPrintJSON(i interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(i); err != nil {
		Fatal(err)
	}
}
