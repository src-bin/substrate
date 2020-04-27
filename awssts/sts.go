package awssts

import (
	"github.com/aws/aws-sdk-go/service/sts"
)

const InvalidClientTokenId = "InvalidClientTokenId"

func GetCallerIdentity(svc *sts.STS) (*sts.GetCallerIdentityOutput, error) {
	return svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
}
