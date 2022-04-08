package awsorgs

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsutil"
)

const (
	ALL                               = "ALL"
	AlreadyInOrganizationException    = "AlreadyInOrganizationException"
	AWSOrganizationsNotInUseException = "AWSOrganizationsNotInUseException"
	TooManyRequestsException          = "TooManyRequestsException"
)

func CreateOrganization(svc *organizations.Organizations) (*organizations.Organization, error) {
	in := &organizations.CreateOrganizationInput{
		FeatureSet: aws.String(ALL), // we want service control policies
	}
	out, err := svc.CreateOrganization(in)
	if err != nil {
		return nil, err
	}
	time.Sleep(10e9) // give Organizations time to finish so that subsequent CreateAccount, etc. will work (TODO do it gracefully)
	return out.Organization, nil
}

func DescribeOrganization(svc *organizations.Organizations) (*organizations.Organization, error) {
	for {
		in := &organizations.DescribeOrganizationInput{}
		out, err := svc.DescribeOrganization(in)
		if err != nil {
			if awsutil.ErrorCodeIs(err, TooManyRequestsException) {
				time.Sleep(time.Second) // TODO exponential backoff
				continue
			}
			return nil, err
		}
		return out.Organization, nil
	}
}

func EnableAWSServiceAccess(svc *organizations.Organizations, principal string) error {
	in := &organizations.EnableAWSServiceAccessInput{
		ServicePrincipal: aws.String(principal),
	}
	_, err := svc.EnableAWSServiceAccess(in)
	return err
}

func Root(svc *organizations.Organizations) (*organizations.Root, error) {
	roots, err := listRoots(svc)
	if err != nil {
		return nil, err
	}
	if len(roots) != 1 {
		return nil, errors.New("ListRoots responded with more than one Root which AWS says is impossible")
	}
	return roots[0], nil
}

func listRoots(svc *organizations.Organizations) (roots []*organizations.Root, err error) {
	var nextToken *string
	for {
		in := &organizations.ListRootsInput{NextToken: nextToken}
		out, err := svc.ListRoots(in)
		if err != nil {
			return nil, err
		}
		roots = append(roots, out.Roots...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
