package regions

import (
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const Filename = "substrate.regions"

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

	if _, err := os.Stat(Filename); os.IsNotExist(err) {
		regions := []string{}
		for _, region := range All() {
			if !IsBeingAvoided(region) {
				regions = append(regions, region)
			}
		}
		if err := ioutil.WriteFile(
			Filename,
			fileutil.FromLines(regions),
			0666,
		); err != nil {
			return nil, err
		}
	}

	regions, err := ui.EditFile(
		Filename,
		"your Substrate-managed infrastructure is currently configured to use the following AWS regions:",
		"remove regions you don't want to use or add regions you wish to expand into, one per line",
	)
	if err != nil {
		return nil, err
	}

	return regions, nil
}

func Selected() []string {
	b, err := fileutil.ReadFile(Filename)
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
