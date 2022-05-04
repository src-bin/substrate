package awssts

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/ui"
	"golang.org/x/crypto/ssh/terminal"
)

func PrintCredentials(format *cmdutil.SerializationFormat, credentials aws.Credentials) {
	switch format.String() {
	case cmdutil.SerializationFormatEnv:
		PrintCredentialsEnv(credentials)
	case cmdutil.SerializationFormatExport:
		PrintCredentialsExport(credentials)
	case cmdutil.SerializationFormatExportWithHistory:
		PrintCredentialsExportWithHistory(credentials)
	case cmdutil.SerializationFormatJSON:
		PrintCredentialsJSON(credentials)
	default:
		ui.Fatalf("-format=%q not supported", format)
	}
}

func PrintCredentialsEnv(credentials aws.Credentials) {
	fmt.Printf(
		"AWS_ACCESS_KEY_ID=%q\nAWS_SECRET_ACCESS_KEY=%q\nAWS_SESSION_TOKEN=%q\n",
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
	)
}

func PrintCredentialsExport(credentials aws.Credentials) {
	fmt.Printf(
		" export AWS_ACCESS_KEY_ID=%q AWS_SECRET_ACCESS_KEY=%q AWS_SESSION_TOKEN=%q\n",
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
	)
}

func PrintCredentialsExportWithHistory(credentials aws.Credentials) {
	if terminal.IsTerminal(1) {
		ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	}
	fmt.Printf(
		` export OLD_AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" AWS_ACCESS_KEY_ID=%q OLD_AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" AWS_SECRET_ACCESS_KEY=%q OLD_AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" AWS_SESSION_TOKEN=%q; alias unassume-role='AWS_ACCESS_KEY_ID="$OLD_AWS_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$OLD_AWS_SECRET_ACCESS_KEY" AWS_SESSION_TOKEN="$OLD_AWS_SESSION_TOKEN"; unset OLD_AWS_ACCESS_KEY_ID OLD_AWS_SECRET_ACCESS_KEY OLD_AWS_SESSION_TOKEN'
`,
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
	)
}

func PrintCredentialsJSON(credentials aws.Credentials) {
	ui.PrettyPrintJSON(struct {
		aws.Credentials
		Version int
	}{credentials, 1})
}
