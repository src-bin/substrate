package awsec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeSubnets(svc *ec2.EC2, vpcId string) ([]*ec2.Subnet, error) {
	in := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{&ec2.Filter{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcId)},
		}},
	}
	//log.Print(in)
	out, err := svc.DescribeSubnets(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Subnets, nil
}
