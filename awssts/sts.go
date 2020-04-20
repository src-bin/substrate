package awssts

import (
	"log"

	"github.com/aws/aws-sdk-go/service/sts"
)

func GetCallerIdentity(svc *sts.STS) *sts.GetCallerIdentityOutput {
	in := &sts.GetCallerIdentityInput{}
	out, err := svc.GetCallerIdentity(in)
	if err != nil {
		log.Fatal(err)
	}
	return out
}
