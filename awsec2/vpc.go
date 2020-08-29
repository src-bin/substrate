package awsec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeSecurityGroups(svc *ec2.EC2, vpcId, name string) ([]*ec2.SecurityGroup, error) {
	in := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(name)},
			},
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
		},
	}
	//log.Print(in)
	out, err := svc.DescribeSecurityGroups(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.SecurityGroups, nil
}

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

func DescribeVpcs(svc *ec2.EC2, environment, quality string) ([]*ec2.Vpc, error) {
	in := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:Environment"),
				Values: []*string{aws.String(environment)},
			},
			&ec2.Filter{
				Name:   aws.String("tag:Quality"),
				Values: []*string{aws.String(quality)},
			},
		},
	}
	//log.Print(in)
	out, err := svc.DescribeVpcs(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Vpcs, nil
}
