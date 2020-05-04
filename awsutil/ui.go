package awsutil

import (
	"log"

	"github.com/src-bin/substrate/ui"
)

func ReadAccessKeyFromStdin() (string, string) {
	accessKeyId, err := ui.Prompt("AWS access key ID:")
	if err != nil {
		log.Fatal(err)
	}
	secretAccessKey, err := ui.Prompt("AWS secret access key:")
	if err != nil {
		log.Fatal(err)
	}
	return accessKeyId, secretAccessKey
}
