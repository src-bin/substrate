package awssts

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/ui"
)

const InvalidClientTokenId = "InvalidClientTokenId"

func AssumeRole(svc *sts.STS, roleArn, sessionName string, durationSeconds int) (*sts.AssumeRoleOutput, error) {
	if sessionName == "" {
		sessionName = filepath.Base(os.Args[0])
	}
	ui.Printf("assuming role %s for %s", roleArn, sessionName)
	return svc.AssumeRole(&sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(int64(durationSeconds)), // can't go longer than the default of 3600 when chaining AssumeRole
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(sessionName),
	})
}

func Export(out *sts.AssumeRoleOutput, err error) {
	if err != nil {
		ui.Print(err)
		return
	}
	ui.Print("paste this into a shell to set environment variables (taking care to preserve the leading space):")
	fmt.Printf(
		` export OLD_AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" AWS_ACCESS_KEY_ID=%q OLD_AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" AWS_SECRET_ACCESS_KEY=%q OLD_AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" AWS_SESSION_TOKEN=%q; alias unassume-role='AWS_ACCESS_KEY_ID="$OLD_AWS_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$OLD_AWS_SECRET_ACCESS_KEY" AWS_SESSION_TOKEN="$OLD_AWS_SESSION_TOKEN"; unset OLD_AWS_ACCESS_KEY_ID OLD_AWS_SECRET_ACCESS_KEY OLD_AWS_SESSION_TOKEN'
`,
		aws.StringValue(out.Credentials.AccessKeyId),
		aws.StringValue(out.Credentials.SecretAccessKey),
		aws.StringValue(out.Credentials.SessionToken),
	)
}

func GetCallerIdentity(svc *sts.STS) (*sts.GetCallerIdentityOutput, error) {
	return svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
}

func MustGetCallerIdentity(svc *sts.STS) *sts.GetCallerIdentityOutput {
	callerIdentity, err := GetCallerIdentity(svc)
	if err != nil {
		log.Fatal(err)
	}
	return callerIdentity
}
