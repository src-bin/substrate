package awsorgs

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
)

const (
	ALL                               = "ALL"
	AlreadyInOrganizationException    = "AlreadyInOrganizationException"
	AWSOrganizationsNotInUseException = "AWSOrganizationsNotInUseException"
)

func CreateOrganization(svc *organizations.Organizations) (*organizations.Organization, error) {
	in := &organizations.CreateOrganizationInput{
		FeatureSet: aws.String(ALL), // we want service control policies
	}
	out, err := svc.CreateOrganization(in)
	return out.Organization, err
}

func DescribeOrganization(svc *organizations.Organizations) (*organizations.Organization, error) {
	in := &organizations.DescribeOrganizationInput{}
	out, err := svc.DescribeOrganization(in)
	return out.Organization, err
}

func Root(svc *organizations.Organizations) *organizations.Root {
	roots := listRoots(svc)
	if len(roots) != 1 {
		log.Fatal("ListRoots responded with more than one Root which AWS says is impossible")
	}
	return roots[0]
}

func listRoots(svc *organizations.Organizations) (roots []*organizations.Root) {
	var nextToken *string
	for {
		in := &organizations.ListRootsInput{NextToken: nextToken}
		out, err := svc.ListRoots(in)
		if err != nil {
			log.Fatal(err)
		}
		roots = append(roots, out.Roots...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
