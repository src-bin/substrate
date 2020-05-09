package awsutil

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func BlacklistedRegions() []string {
	return []string{
		"ap-east-1",      // IAM seems broken
		"ap-south-1",     // VPC quotas make me sad
		"ap-southeast-1", // VPC quotas make me sad
		"ap-southeast-2", // VPC quotas make me sad
		"eu-central-1",   // VPC quotas make me sad
		"eu-north-1",     // Service Quotas seem broken
		"eu-west-3",      // VPC quotas make me sad
		"me-south-1",     // IAM seems broken
		"sa-east-1",      // VPC quotas make me sad
	} // this list must remain sorted
}

func IsBlacklistedRegion(region string) bool {
	blacklist := BlacklistedRegions()
	i := sort.SearchStrings(blacklist, region)
	return i < len(blacklist) && blacklist[i] == region
}

func IsRegion(region string) bool {
	_, ok := regions()[region]
	return ok
}

func Regions() []string {
	m := regions()
	s := make([]string, 0, len(m))
	for region, _ := range m {
		s = append(s, region)
	}
	sort.Strings(s)
	return s
}

func regions() map[string]endpoints.Region {
	return endpoints.AwsPartition().Regions()
}
