package ui

import (
	"encoding/json"
	"io"
)

func PrettyPrintJSON(w io.Writer, i interface{}) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(i); err != nil {
		Fatal(err)
	}
}
