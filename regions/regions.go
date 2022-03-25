package regions

import (
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const (
	DefaultRegionFilename = "substrate.default-region"
	RegionsFilename       = "substrate.regions"

	Global = "global" // special value used in the same place as region sometimes
)

func All() []string {
	m := all()
	ss := make([]string, 0, len(m))
	for region, _ := range m {
		ss = append(ss, region)
	}
	sort.Strings(ss)
	return ss
}

func Avoiding() []string {
	return []string{
		"ap-east-1", // IAM seems broken
		//"ap-south-1",     // VPC quotas make me sad
		//"ap-southeast-1", // VPC quotas make me sad
		//"ap-southeast-2", // VPC quotas make me sad
		//"eu-central-1",   // VPC quotas make me sad
		"eu-north-1", // Service Quotas seem broken
		//"eu-west-3",      // VPC quotas make me sad
		"me-south-1", // IAM seems broken
		//"sa-east-1",      // VPC quotas make me sad
	} // this list must remain sorted
}

func Default() string {
	region, err := ui.PromptFile(
		DefaultRegionFilename,
		"what region is your default for hosting shared resources e.g. the S3 bucket that stores your CloudTrail logs?",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !IsRegion(region) {
		log.Fatalf("%s is not an AWS region", region)
	}
	if !printedDefaultRegion {
		ui.Printf("using region %s as your default", region)
		printedDefaultRegion = true
	}
	return region
}

func DefaultNoninteractive() (string, error) {
	pathname, err := fileutil.PathnameInParents(DefaultRegionFilename)
	if err != nil {
		return "", err
	}
	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		return "", err
	}
	return fileutil.Tidy(b), nil
}

func IsBeingAvoided(region string) bool {
	avoiding := Avoiding()
	i := sort.SearchStrings(avoiding, region)
	return i < len(avoiding) && avoiding[i] == region
}

func IsRegion(region string) bool {
	_, ok := all()[region]
	return ok
}

func Select() ([]string, error) {

	// We used to begin by initializing the list of regions with all of them.
	// This was problematic for two reasons:
	// 1. It encouraged folks to enable them all which takes forever and
	//    potentially costs a lot just for the NAT Gateways.
	// 2. It breaks the signal to the various levels of interactivity.
	/*
		if !fileutil.Exists(RegionsFilename) {
			regions := []string{}
			for _, region := range All() {
				if !IsBeingAvoided(region) {
					regions = append(regions, region)
				}
			}
			if err := ioutil.WriteFile(
				RegionsFilename,
				fileutil.FromLines(regions),
				0666,
			); err != nil {
				return nil, err
			}
		}
	*/

	regions, err := ui.EditFile(
		RegionsFilename,
		"your Substrate-managed infrastructure is currently configured to use the following AWS regions:",
		// "remove regions you don't want to use or add regions you wish to expand into, one per line",
		"add regions you wish to expand into, one per line",
	)
	if err != nil {
		return nil, err
	}

	return regions, nil
}

func Selected() []string {
	b, err := fileutil.ReadFile(RegionsFilename)
	if err != nil {
		log.Fatal(err)
	}
	ss := fileutil.ToLines(b)
	sort.Strings(ss)
	return ss
}

func all() map[string]endpoints.Region {
	return endpoints.AwsPartition().Regions()
}

var printedDefaultRegion bool
