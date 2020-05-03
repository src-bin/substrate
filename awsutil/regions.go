package awsutil

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

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
