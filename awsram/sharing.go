package awsram

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ram"
)

func EnableSharingWithAwsOrganization(svc *ram.RAM) error {
	in := &ram.EnableSharingWithAwsOrganizationInput{}
	out, err := svc.EnableSharingWithAwsOrganization(in)
	if err == nil && out != nil && !aws.BoolValue(out.ReturnValue) {
		err = errors.New("EnableSharingWithAwsOrganization received ReturnValue: false")
	}
	return err
}
