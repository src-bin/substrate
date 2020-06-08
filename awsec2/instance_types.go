package awsec2

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeInstanceTypeOfferings(svc *ec2.EC2) (offerings []*ec2.InstanceTypeOffering, err error) {
	var nextToken *string
	for {
		in := &ec2.DescribeInstanceTypeOfferingsInput{
			NextToken: nextToken,
		}
		out, err := svc.DescribeInstanceTypeOfferings(in)
		if err != nil {
			return nil, err
		}
		offerings = append(offerings, out.InstanceTypeOfferings...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
