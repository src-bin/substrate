package awsutil

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func BlacklistedRegions() []string {
	return []string{
		"ap-east-1",
		"me-south-1",
	} // this list must remain sorted
}

func IsBlacklistedRegion(region string) bool {
	blacklist := BlacklistedRegions()
	return sort.SearchStrings(blacklist, region) != len(blacklist)
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
