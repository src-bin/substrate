package awsec2

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	Amazon       = "amazon"
	AmazonLinux2 = "amzn2-ami-*-gp2"

	ARM    = "arm64"
	X86_64 = "x86_64"
)

func DescribeImages(svc *ec2.EC2, arch, name, owner string) ([]*ec2.Image, error) {
	in := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("architecture"),
				Values: []*string{aws.String(arch)},
			},
			&ec2.Filter{
				Name:   aws.String("name"),
				Values: []*string{aws.String(name)},
			},
		},
		Owners: []*string{aws.String(owner)},
	}
	//log.Print(in)
	out, err := svc.DescribeImages(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Images, nil
}

func LatestAmazonLinux2AMI(svc *ec2.EC2, arch string) (*ec2.Image, error) {
	images, err := DescribeImages(svc, arch, AmazonLinux2, Amazon)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("Amazon Linux 2 AMI for %s not found", arch)
	}
	sort.Slice(images, func(i, j int) bool {
		return aws.StringValue(images[j].CreationDate) < aws.StringValue(images[i].CreationDate) // descending
	})
	return images[0], nil
}

func RunInstances(svc *ec2.EC2, imageId, instanceType, subnetId string, tags []*ec2.Tag) (*ec2.Reservation, error) {
	in := &ec2.RunInstancesInput{
		DryRun: aws.Bool(true), // XXX
		//IamInstanceProfile
		ImageId:      aws.String(imageId),
		InstanceType: aws.String(instanceType),
		//KeyName
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
		//SecurityGroupIds
		SubnetId: aws.String(subnetId),
		TagSpecifications: []*ec2.TagSpecification{&ec2.TagSpecification{
			ResourceType: aws.String("instance"),
			Tags:         tags,
		}},
		//UserData
	}
	//log.Print(in)
	reservation, err := svc.RunInstances(in)
	if err != nil {
		return nil, err
	}
	//log.Print(reservation)
	return reservation, nil
}
