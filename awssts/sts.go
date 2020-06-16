package awssts

import (
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/ui"
)

const InvalidClientTokenId = "InvalidClientTokenId"

func AssumeRole(svc *sts.STS, roleArn, sessionName string) (*sts.AssumeRoleOutput, error) {
	if sessionName == "" {
		sessionName = filepath.Base(os.Args[0])
	}
	ui.Printf("assuming role %s for %s", roleArn, sessionName)
	return svc.AssumeRole(&sts.AssumeRoleInput{
		// DurationSeconds: aws.Int64(12 * 60 * 60), // can't go longer than the default of 3600 when chaining AssumeRole
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
	ui.Printf(
		" export AWS_ACCESS_KEY_ID=%q AWS_SECRET_ACCESS_KEY=%q AWS_SESSION_TOKEN=%q",
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
