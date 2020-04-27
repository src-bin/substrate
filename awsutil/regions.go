package awsutil

import "github.com/aws/aws-sdk-go/aws/endpoints"

func IsRegion(region string) bool {
	_, ok := Regions()[region]
	return ok
}

func Regions() map[string]endpoints.Region {
	return endpoints.AwsPartition().Regions()
}
