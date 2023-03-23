package awsutil

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/ui"
	"golang.org/x/crypto/ssh/terminal"
)

func PrintCredentials(format *cmdutil.SerializationFormat, creds aws.Credentials) {
	// Check if we're using Fish as our shell. If so, we have to use it's unique and special syntax for variables
	isFish := CheckForFish()

	switch format.String() {
	case cmdutil.SerializationFormatEnv:
		PrintCredentialsEnv(creds, isFish)
	case cmdutil.SerializationFormatExport:
		PrintCredentialsExport(creds, isFish)
	case cmdutil.SerializationFormatExportWithHistory:
		PrintCredentialsExportWithHistory(creds, isFish)
	case cmdutil.SerializationFormatJSON:
		PrintCredentialsJSON(creds)
	default:
		ui.Fatalf("-format %q not supported", format)
	}
}

// CheckForFish finds the name of Substrate's parent process (ppid) and if it's the fish shell, return true.
func CheckForFish() bool {
	parentProcess, _ := process.NewProcess(int32(os.Getppid()))
	parentName, _ := parentProcess.Name()

	if parentName == "fish" {
		return true
	} else {
		return false
	}
}

func PrintCredentialsEnv(creds aws.Credentials, isFish bool) {
	if isFish {
		fmt.Printf(
			"set AWS_ACCESS_KEY_ID %q\nset AWS_SECRET_ACCESS_KEY %q\nset AWS_SESSION_TOKEN %q\n",
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	} else {
		fmt.Printf(
			"AWS_ACCESS_KEY_ID=%q\nAWS_SECRET_ACCESS_KEY=%q\nAWS_SESSION_TOKEN=%q\n",
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	}
}

func PrintCredentialsExport(creds aws.Credentials, isFish bool) {
	if isFish {
		fmt.Printf(
			" set -x AWS_ACCESS_KEY_ID %q; set -x AWS_SECRET_ACCESS_KEY %q; set -x AWS_SESSION_TOKEN %q\n",
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	} else {
		fmt.Printf(
			" export AWS_ACCESS_KEY_ID=%q AWS_SECRET_ACCESS_KEY=%q AWS_SESSION_TOKEN=%q\n",
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	}
}

func PrintCredentialsExportWithHistory(creds aws.Credentials, isFish bool) {
	if terminal.IsTerminal(1) {
		ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	}

	if isFish {
		fmt.Printf(
			` set -x OLD_AWS_ACCESS_KEY_ID "$AWS_ACCESS_KEY_ID"; set -x AWS_ACCESS_KEY_ID %q; set -x OLD_AWS_SECRET_ACCESS_KEY "$AWS_SECRET_ACCESS_KEY"; set -x AWS_SECRET_ACCESS_KEY %q; set -x OLD_AWS_SESSION_TOKEN "$AWS_SESSION_TOKEN"; set -x AWS_SESSION_TOKEN %q; alias unassume-role 'set AWS_ACCESS_KEY_ID "$OLD_AWS_ACCESS_KEY_ID"; set AWS_SECRET_ACCESS_KEY "$OLD_AWS_SECRET_ACCESS_KEY"; set AWS_SESSION_TOKEN "$OLD_AWS_SESSION_TOKEN"; set -e OLD_AWS_ACCESS_KEY_ID; set -e OLD_AWS_SECRET_ACCESS_KEY; set -e OLD_AWS_SESSION_TOKEN'
`,
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	} else {
		fmt.Printf(
			` export OLD_AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" AWS_ACCESS_KEY_ID=%q OLD_AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" AWS_SECRET_ACCESS_KEY=%q OLD_AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" AWS_SESSION_TOKEN=%q; alias unassume-role='AWS_ACCESS_KEY_ID="$OLD_AWS_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$OLD_AWS_SECRET_ACCESS_KEY" AWS_SESSION_TOKEN="$OLD_AWS_SESSION_TOKEN"; unset OLD_AWS_ACCESS_KEY_ID OLD_AWS_SECRET_ACCESS_KEY OLD_AWS_SESSION_TOKEN'
`,
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)
	}
}

func PrintCredentialsJSON(creds aws.Credentials) {
	jsonutil.PrettyPrint(os.Stdout, struct {
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
