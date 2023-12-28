package cmdutil

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/ui"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	AWS_ACCESS_KEY_ID                = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY            = "AWS_SECRET_ACCESS_KEY"
	AWS_SESSION_TOKEN                = "AWS_SESSION_TOKEN"
	SUBSTRATE_CREDENTIALS_EXPIRATION = "SUBSTRATE_CREDENTIALS_EXPIRATION"
)

func PrintCredentials(format Format, creds aws.Credentials) {
	// Check if we're using Fish as our shell. If so, we have to use it's unique and special syntax for variables
	isFish := CheckForFish()

	switch format {
	case FormatEnv:
		PrintCredentialsEnv(creds, isFish)
	case FormatExport:
		PrintCredentialsExport(creds, isFish)
	case FormatExportWithHistory:
		PrintCredentialsExportWithHistory(creds, isFish)
	case FormatJSON:
		PrintCredentialsJSON(creds)
	default:
		ui.Fatal(FormatFlagError(format))
	}
}

// CheckForFish finds the name of Substrate's parent process (ppid) and if it's the fish shell, return true.
func CheckForFish() bool {
	parentName, err := ParentProcessName()
	// fmt.Fprintf(os.Stderr, "parentName: %s", parentName)
	if err != nil {
		return false
	}

	if strings.Contains(parentName, "fish") {
		return true
	} else {
		return false
	}
}

func PrintCredentialsEnv(creds aws.Credentials, isFish bool) {
	if isFish {
		fmt.Printf(
			"set %s %q\nset %s %q\nset %s %q\nset %s %q\n",
			AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
		)
	} else {
		fmt.Printf(
			"%s=%q\n%s=%q\n%s=%q\n%s=%q\n",
			AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
		)
	}
}

func PrintCredentialsExport(creds aws.Credentials, isFish bool) {
	if isFish {
		fmt.Printf(
			" set -x %s %q; set -x %s %q; set -x %s %q; set -x %s %q\n",
			AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
		)
	} else {
		fmt.Printf(
			" export %s=%q %s=%q %s=%q %s=%q\n",
			AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
		)
	}
}

func PrintCredentialsExportWithHistory(creds aws.Credentials, isFish bool) {
	if terminal.IsTerminal(1) {
		ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	}

	if isFish {
		fmt.Printf(
			` set -x OLD_%s "$%s"; set -x %s %q; set -x OLD_%s "$%s"; set -x %s %q; set -x OLD_%s "$%s"; set -x %s %q; set -x OLD_%s "$%s"; set -x %s %q; alias unassume-role 'set %s "$OLD_%s"; set %s "$OLD_%s"; set %s "$OLD_%s"; set %s "$OLD_%s"; set -e OLD_%s; set -e OLD_%s; set -e OLD_%s; set -e OLD_%s'
`,
			AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN, AWS_SESSION_TOKEN, AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
			AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID,
			AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY,
			AWS_SESSION_TOKEN, AWS_SESSION_TOKEN,
			SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION,
			AWS_ACCESS_KEY_ID,
			AWS_SECRET_ACCESS_KEY,
			AWS_SESSION_TOKEN,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
		)
	} else {
		fmt.Printf(
			` export OLD_%s="$%s" %s=%q OLD_%s="$%s" %s=%q OLD_%s="$%s" %s=%q OLD_%s="$%s" %s=%q; alias unassume-role='%s="$OLD_%s" %s="$OLD_%s" %s="$OLD_%s" %s="$OLD_%s"; unset OLD_%s OLD_%s OLD_%s OLD_%s'
`,
			AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID,
			creds.AccessKeyID,
			AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY,
			creds.SecretAccessKey,
			AWS_SESSION_TOKEN, AWS_SESSION_TOKEN, AWS_SESSION_TOKEN,
			creds.SessionToken,
			SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION,
			creds.Expires.Format(time.RFC3339),
			AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID,
			AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY,
			AWS_SESSION_TOKEN, AWS_SESSION_TOKEN,
			SUBSTRATE_CREDENTIALS_EXPIRATION, SUBSTRATE_CREDENTIALS_EXPIRATION,
			AWS_ACCESS_KEY_ID,
			AWS_SECRET_ACCESS_KEY,
			AWS_SESSION_TOKEN,
			SUBSTRATE_CREDENTIALS_EXPIRATION,
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

func Setenv(creds aws.Credentials) (err error) {
	if err = os.Setenv(AWS_ACCESS_KEY_ID, creds.AccessKeyID); err != nil {
		return
	}
	if err = os.Setenv(AWS_SECRET_ACCESS_KEY, creds.SecretAccessKey); err != nil {
		return
	}
	if creds.SessionToken == "" {
		err = os.Unsetenv(AWS_SESSION_TOKEN)
	} else {
		err = os.Setenv(AWS_SESSION_TOKEN, creds.SessionToken)
	}
	if err != nil {
		return
	}
	err = os.Setenv(SUBSTRATE_CREDENTIALS_EXPIRATION, creds.Expires.Format(time.RFC3339))
	return
}
