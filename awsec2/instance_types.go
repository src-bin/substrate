package awsec2

import (
	"github.com/aws/aws-sdk-go/aws"
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

func DescribeInstanceTypes(svc *ec2.EC2, types []string) (infos []*ec2.InstanceTypeInfo, err error) {
	var nextToken *string
	for {
		in := &ec2.DescribeInstanceTypesInput{
			InstanceTypes: aws.StringSlice(types),
			NextToken:     nextToken,
		}
		out, err := svc.DescribeInstanceTypes(in)
		if err != nil {
			return nil, err
		}
		infos = append(infos, out.InstanceTypes...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
