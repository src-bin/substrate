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
