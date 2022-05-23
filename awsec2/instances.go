package awsec2

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const (
	Amazon       = "amazon"
	AmazonLinux2 = "amzn2-ami-*-gp2" // TODO should this be updated to say gp3 or ...?

	ARM    = types.ArchitectureTypeArm64
	X86_64 = types.ArchitectureTypeX8664

	InvalidLaunchTemplateName_NotFound = "InvalidLaunchTemplateName.NotFound"
	Unsupported                        = "Unsupported"
	UnsupportedOperation               = "UnsupportedOperation"
)

type (
	ArchitectureType   = types.ArchitectureType
	Filter             = types.Filter
	Image              = types.Image
	Instance           = types.Instance
	KeyPairInfo        = types.KeyPairInfo
	RunInstancesOutput = ec2.RunInstancesOutput
	Tag                = types.Tag
)

func DescribeImages(
	ctx context.Context,
	cfg *awscfg.Config,
	arch ArchitectureType,
	name, owner string,
) ([]Image, error) {
	out, err := cfg.EC2().DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("architecture"),
				Values: []string{string(arch)},
			},
			{
				Name:   aws.String("name"),
				Values: []string{name},
			},
		},
		Owners: []string{owner},
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Images, nil
}

func DescribeInstances(
	ctx context.Context,
	cfg *awscfg.Config,
	filters []Filter,
) ([]Instance, error) {
	out, err := cfg.EC2().DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: filters})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	var instances []types.Instance
	for _, reservation := range out.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances, nil
}

func DescribeKeyPairs(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) ([]KeyPairInfo, error) {
	out, err := cfg.EC2().DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{name},
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.KeyPairs, nil
}

func ImportKeyPair(
	ctx context.Context,
	cfg *awscfg.Config,
	name, publicKeyMaterial string,
) (*ec2.ImportKeyPairOutput, error) {
	out, err := cfg.EC2().ImportKeyPair(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(name),
		PublicKeyMaterial: []byte(publicKeyMaterial),
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out, nil
}

func LatestAmazonLinux2AMI(
	ctx context.Context,
	cfg *awscfg.Config,
	arch ArchitectureType,
) (*Image, error) {
	images, err := DescribeImages(ctx, cfg, arch, AmazonLinux2, Amazon)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("Amazon Linux 2 AMI for %s not found", arch)
	}
	sort.Slice(images, func(i, j int) bool {
		return aws.ToString(images[j].CreationDate) < aws.ToString(images[i].CreationDate) // descending
	})
	image := images[0] // don't leak the slice
	return &image, nil
}

func RunInstance(
	ctx context.Context,
	cfg *awscfg.Config,
	iamInstanceProfile, imageId string,
	instanceType InstanceType,
	keyName, launchTemplateName string,
	rootVolumeSize int32,
	securityGroupId, subnetId string,
	tags []Tag,
) (reservation *RunInstancesOutput, err error) {
	in := &ec2.RunInstancesInput{
		BlockDeviceMappings: []types.BlockDeviceMapping{{
			DeviceName: aws.String("/dev/xvda"),
			Ebs: &types.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				VolumeSize:          aws.Int32(rootVolumeSize),
				VolumeType:          types.VolumeTypeGp3,
			},
		}},
		EbsOptimized: aws.Bool(true),
		IamInstanceProfile: &types.IamInstanceProfileSpecification{
			Name: aws.String(iamInstanceProfile),
		},
		ImageId:      aws.String(imageId),
		InstanceType: instanceType,
		KeyName:      aws.String(keyName),
		LaunchTemplate: &types.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(launchTemplateName),
		},
		MaxCount: aws.Int32(1),
		MetadataOptions: &types.InstanceMetadataOptionsRequest{
			HttpEndpoint:         types.InstanceMetadataEndpointStateEnabled,
			HttpProtocolIpv6:     types.InstanceMetadataProtocolStateEnabled,
			HttpTokens:           types.HttpTokensStateRequired,
			InstanceMetadataTags: types.InstanceMetadataTagsStateEnabled,
		},
		MinCount:         aws.Int32(1),
		SecurityGroupIds: []string{securityGroupId},
		SubnetId:         aws.String(subnetId),
		TagSpecifications: []types.TagSpecification{{
			ResourceType: types.ResourceTypeInstance,
			Tags:         tags,
		}},
	}
	client := cfg.EC2()
	for {
		reservation, err = client.RunInstances(ctx, in)
		if awsutil.ErrorCodeIs(err, InvalidLaunchTemplateName_NotFound) {
			in.LaunchTemplate = nil
		} else if awsutil.ErrorCodeIs(err, Unsupported) {
			if in.EbsOptimized == nil {
				break // we've already tried unsetting this and it's still failing, so fail
			}
			in.EbsOptimized = nil
		} else if awsutil.ErrorCodeIs(err, UnsupportedOperation) {
			in.MetadataOptions.HttpProtocolIpv6 = types.InstanceMetadataProtocolStateDisabled
		} else {
			break
		}
	}
	return
}

func TerminateInstance(
	ctx context.Context,
	cfg *awscfg.Config,
	instanceId string,
) error {
	_, err := cfg.EC2().TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return err
	}
	//log.Print(out)
	return nil
}
