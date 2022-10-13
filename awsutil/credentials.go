package awsutil

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/ui"
	"golang.org/x/crypto/ssh/terminal"
)

func PrintCredentials(format *cmdutil.SerializationFormat, creds aws.Credentials) {
	switch format.String() {
	case cmdutil.SerializationFormatEnv:
		PrintCredentialsEnv(creds)
	case cmdutil.SerializationFormatExport:
		PrintCredentialsExport(creds)
	case cmdutil.SerializationFormatExportWithHistory:
		PrintCredentialsExportWithHistory(creds)
	case cmdutil.SerializationFormatJSON:
		PrintCredentialsJSON(creds)
	default:
		ui.Fatalf("-format %q not supported", format)
	}
}

func PrintCredentialsEnv(creds aws.Credentials) {
	fmt.Printf(
		"AWS_ACCESS_KEY_ID=%q\nAWS_SECRET_ACCESS_KEY=%q\nAWS_SESSION_TOKEN=%q\n",
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)
}

func PrintCredentialsExport(creds aws.Credentials) {
	fmt.Printf(
		" export AWS_ACCESS_KEY_ID=%q AWS_SECRET_ACCESS_KEY=%q AWS_SESSION_TOKEN=%q\n",
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)
}

func PrintCredentialsExportWithHistory(creds aws.Credentials) {
	if terminal.IsTerminal(1) {
		ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	}
	fmt.Printf(
		` export OLD_AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" AWS_ACCESS_KEY_ID=%q OLD_AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" AWS_SECRET_ACCESS_KEY=%q OLD_AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" AWS_SESSION_TOKEN=%q; alias unassume-role='AWS_ACCESS_KEY_ID="$OLD_AWS_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$OLD_AWS_SECRET_ACCESS_KEY" AWS_SESSION_TOKEN="$OLD_AWS_SESSION_TOKEN"; unset OLD_AWS_ACCESS_KEY_ID OLD_AWS_SECRET_ACCESS_KEY OLD_AWS_SESSION_TOKEN'
`,
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)
}

func PrintCredentialsJSON(creds aws.Credentials) {
	ui.PrettyPrintJSON(struct {
		AccessKeyId     string // must be "Id" not "ID" for AWS SDK credential_process
		SecretAccessKey string
		SessionToken    string
		Expiration      string // must be "Expiration" not "Expires" for AWS SDK credential_process
		Version         int
	}{
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
		creds.Expires.Format(time.RFC3339),
		1,
	})
}
