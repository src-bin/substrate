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
	if err != nil {
		return nil, err
	}
	return out.Organization, nil
}

func DescribeOrganization(svc *organizations.Organizations) (*organizations.Organization, error) {
	in := &organizations.DescribeOrganizationInput{}
	out, err := svc.DescribeOrganization(in)
	if err != nil {
		return nil, err
	}
	return out.Organization, nil
}

func Root(svc *organizations.Organizations) *organizations.Root {
	ch := listRoots(svc)
	root := <-ch
	if root2 := <-ch; root2 != nil {
		log.Fatal("ListRoots responded with more than one Root which AWS says is impossible")
	}
	return root
}

func listRoots(svc *organizations.Organizations) <-chan *organizations.Root {
	ch := make(chan *organizations.Root)
	go func(chan<- *organizations.Root) {
		var nextToken *string
		for {
			in := &organizations.ListRootsInput{NextToken: nextToken}
			out, err := svc.ListRoots(in)
			if err != nil {
				log.Fatal(err)
			}
			for _, root := range out.Roots {
				ch <- root
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		close(ch)
	}(ch)
	return ch
}
