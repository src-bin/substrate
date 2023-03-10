package jsonutil

import (
	"encoding/json"
	"io"

	"github.com/src-bin/substrate/ui"
)

func PrettyPrint(w io.Writer, i interface{}) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(i); err != nil {
		ui.Fatal(err)
	}
}
