package randutil

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/src-bin/substrate/ui"
)

func String() string {
	b := make([]byte, 48) // 48 binary bytes makes 64 base64-encoded bytes
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		ui.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
