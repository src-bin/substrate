package awsec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeAvailabilityZones(svc *ec2.EC2, region string) ([]*ec2.AvailabilityZone, error) {
	in := &ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{&ec2.Filter{
			Name:   aws.String("region-name"),
			Values: []*string{aws.String(region)},
		}},
	}
	out, err := svc.DescribeAvailabilityZones(in)
	if err != nil {
		return nil, err
	}
	return out.AvailabilityZones, nil
}
