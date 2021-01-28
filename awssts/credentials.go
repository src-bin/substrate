package awssts

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/ui"
)

const (
	CredentialFormatEnv               = "env"
	CredentialFormatExport            = "export"
	CredentialFormatExportWithHistory = "export-with-history"
	CredentialFormatJSON              = "json"
)

type CredentialFormat struct {
	format string
}

func CredentialFormatFlag() *CredentialFormat {
	f := &CredentialFormat{}
	flag.Var(
		f,
		"format",
		`output format - "export" for exported shell environment variables, "env" for .env files, "json" for IAM API-compatible JSON`,
		// "export-with-history" works but is undocumented because it only really makes sense as used by substrate-assume-role
	)
	return f
}

func (f *CredentialFormat) Print(credentials *sts.Credentials) {
	formats[f.String()](credentials)
}

func (f *CredentialFormat) Set(format string) error {
	if _, ok := formats[format]; !ok {
		return CredentialFormatError(fmt.Sprintf(`-format="%s" not supported`, format))
	}
	f.format = format
	return nil
}

func (f *CredentialFormat) String() string {
	if f.format == "" {
		return CredentialFormatExport
	}
	return f.format
}

type CredentialFormatError string

func (err CredentialFormatError) Error() string {
	return string(err)
}

func PrintCredentialsEnv(credentials *sts.Credentials) {
	fmt.Printf(
		"AWS_ACCESS_KEY_ID=\"%s\"\nAWS_SECRET_ACCESS_KEY=\"%s\"\nAWS_SESSION_TOKEN=\"%s\"\n",
		aws.StringValue(credentials.AccessKeyId),
		aws.StringValue(credentials.SecretAccessKey),
		aws.StringValue(credentials.SessionToken),
	)
}

func PrintCredentialsExport(credentials *sts.Credentials) {
	fmt.Printf(
		" export AWS_ACCESS_KEY_ID=\"%s\" AWS_SECRET_ACCESS_KEY=\"%s\" AWS_SESSION_TOKEN=\"%s\"\n",
		aws.StringValue(credentials.AccessKeyId),
		aws.StringValue(credentials.SecretAccessKey),
		aws.StringValue(credentials.SessionToken),
	)
}

func PrintCredentialsExportWithHistory(credentials *sts.Credentials) {
	ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	fmt.Printf(
		` export OLD_AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" AWS_ACCESS_KEY_ID=%q OLD_AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" AWS_SECRET_ACCESS_KEY=%q OLD_AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" AWS_SESSION_TOKEN=%q; alias unassume-role='AWS_ACCESS_KEY_ID="$OLD_AWS_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$OLD_AWS_SECRET_ACCESS_KEY" AWS_SESSION_TOKEN="$OLD_AWS_SESSION_TOKEN"; unset OLD_AWS_ACCESS_KEY_ID OLD_AWS_SECRET_ACCESS_KEY OLD_AWS_SESSION_TOKEN'
`,
		aws.StringValue(credentials.AccessKeyId),
		aws.StringValue(credentials.SecretAccessKey),
		aws.StringValue(credentials.SessionToken),
	)
}

func PrintCredentialsJSON(credentials *sts.Credentials) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(struct {
		Credentials *sts.Credentials // nested to behave exactly like `aws sts assume-role`
	}{credentials}); err != nil {
		ui.Fatal(err)
	}
}

var formats = map[string]func(*sts.Credentials){
	CredentialFormatEnv:               PrintCredentialsEnv,
	CredentialFormatExport:            PrintCredentialsExport,
	CredentialFormatExportWithHistory: PrintCredentialsExportWithHistory,
	CredentialFormatJSON:              PrintCredentialsJSON,
}
