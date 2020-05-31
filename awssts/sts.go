package awssts

import (
	"log"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/ui"
)

const InvalidClientTokenId = "InvalidClientTokenId"

func AssumeRole(svc *sts.STS, roleArn string) (*sts.AssumeRoleOutput, error) {
	return svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(path.Base(os.Args[0])),
	})
}

func Export(out *sts.AssumeRoleOutput, err error) {
	if err != nil {
		ui.Print(err)
		return
	}
	ui.Printf(
		`export AWS_ACCESS_KEY_ID=%#v AWS_SECRET_ACCESS_KEY=%#v AWS_SESSION_TOKEN=%#v`,
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
