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

func DescribeInstances(
	svc *ec2.EC2,
	filters []*ec2.Filter,
) ([]*ec2.Instance, error) {
	in := &ec2.DescribeInstancesInput{Filters: filters}
	//log.Print(in)
	out, err := svc.DescribeInstances(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	var instances []*ec2.Instance
	for _, reservation := range out.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances, nil
}

func DescribeKeyPairs(svc *ec2.EC2, name string) ([]*ec2.KeyPairInfo, error) {
	in := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(name)},
	}
	//log.Print(in)
	out, err := svc.DescribeKeyPairs(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.KeyPairs, nil
}

func ImportKeyPair(svc *ec2.EC2, name, publicKeyMaterial string) (*ec2.ImportKeyPairOutput, error) {
	in := &ec2.ImportKeyPairInput{
		KeyName:           aws.String(name),
		PublicKeyMaterial: []byte(publicKeyMaterial),
	}
	//log.Print(in)
	out, err := svc.ImportKeyPair(in)
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out, nil
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

func RunInstance(
	svc *ec2.EC2,
	iamInstanceProfile, imageId, instanceType, keyName string,
	rootVolumeSize int,
	securityGroupId, subnetId string,
	tags []*ec2.Tag,
) (*ec2.Reservation, error) {
	in := &ec2.RunInstancesInput{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{&ec2.BlockDeviceMapping{
			DeviceName: aws.String("/dev/xvda"),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				VolumeSize:          aws.Int64(int64(rootVolumeSize)),
				VolumeType:          aws.String("gp3"),
			},
		}},
		//DryRun: aws.Bool(true),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(iamInstanceProfile),
		},
		ImageId:      aws.String(imageId),
		InstanceType: aws.String(instanceType),
		KeyName:      aws.String(keyName),
		MaxCount:     aws.Int64(1),
		MetadataOptions: &ec2.InstanceMetadataOptionsRequest{
			HttpEndpoint:     aws.String("enabled"),
			HttpProtocolIpv6: aws.String("enabled"),
			HttpTokens:       aws.String("required"),
		},
		MinCount:         aws.Int64(1),
		SecurityGroupIds: []*string{aws.String(securityGroupId)},
		SubnetId:         aws.String(subnetId),
		TagSpecifications: []*ec2.TagSpecification{&ec2.TagSpecification{
			ResourceType: aws.String("instance"),
			Tags:         tags,
		}},
	}
	//log.Print(in)
	reservation, err := svc.RunInstances(in)
	if err != nil {
		return nil, err
	}
	//log.Print(reservation)
	return reservation, nil
}

func TerminateInstance(
	svc *ec2.EC2, instanceId string) error {
	in := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(instanceId)},
	}
	//log.Print(in)
	_, err := svc.TerminateInstances(in)
	if err != nil {
		return err
	}
	//log.Print(out)
	return nil
}
