package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/regions"
)

func main() {

	accountId := flag.String("account-number", "", "AWS account number")
	rolename := flag.String("role", "", "AWS IAM role name to assume")
	region := flag.String("region", "", "AWS region in which the service quota should be shown or increased")
	listServices := flag.Bool("list-services", false, "list all services that have service limits to learn their -service-code values")
	listQuotas := flag.Bool("list-quotas", false, "list all service quotes for -service-code to learn their -quota-code values")
	allRegions := flag.Bool("all-regions", false, "show or increase the service quota in all AWS regions")
	quotaCode := flag.String("quota-code", "", "quota code to pass to AWS")
	serviceCode := flag.String("service-code", "", "quota code to pass to AWS")
	desiredValue := flag.Float64("desired-value", 0, "minimum desired value for the service quota")

	flag.Parse()

	if !*allRegions && !regions.IsRegion(*region) {
		log.Fatal("one of -all-regions or a valid -region is required")
	}
	var regionSlice []string
	if *allRegions {
		for _, region := range regions.Selected() {
			regionSlice = append(regionSlice, region)
		}
	} else {
		regionSlice = []string{*region}
	}

	// TODO factor this part out into substrate-assume-role to simplify this tools interface
	sess := awssessions.AssumeRole(
		session.Must(awssessions.NewSession(awssessions.Config{})),
		*accountId,
		*rolename,
	)

	if *listServices {
		var lines []string
		for _, region := range regionSlice {
			for info := range awsservicequotas.ListServices(
				servicequotasNew(sess, region),
			) {
				lines = append(lines, fmt.Sprintf(
					"%-23s %s\n",
					aws.StringValue(info.ServiceCode),
					aws.StringValue(info.ServiceName),
				))
			}
		}
		sort.Strings(lines)
		var previousLine string
		for _, line := range lines {
			if line != previousLine {
				fmt.Print(line)
			}
			previousLine = line
		}
		return
	}

	if *listQuotas {
		if *serviceCode == "" {
			log.Fatal("-service-code is required with -list-quotas")
		}
		for _, region := range regionSlice {
			for quota := range awsservicequotas.ListServiceQuotas(
				servicequotasNew(sess, region),
				*serviceCode,
			) {
				fmt.Printf("%+v\n", quota)
			}
		}
		return
	}

	if *quotaCode == "" || *serviceCode == "" {
		log.Fatal("-quota-code and -service-code are required without -list-services or -list-quotas")
	}
	if *desiredValue > 0 {
		if *allRegions {

			if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
				sess,
				*quotaCode,
				*serviceCode,
				*desiredValue,
			); err != nil {
				log.Fatal(err)
			}

		} else {

			if err := awsservicequotas.EnsureServiceQuota(
				servicequotasNew(sess, *region),
				*quotaCode,
				*serviceCode,
				*desiredValue,
			); err != nil {
				log.Fatal(err)
			}

		}
	} else {

		for _, region := range regionSlice {
			quota, err := awsservicequotas.GetServiceQuota(
				servicequotasNew(sess, region),
				*quotaCode,
				*serviceCode,
			)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%+v\n", quota)
		}

	}

}

func servicequotasNew(sess *session.Session, region string) *servicequotas.ServiceQuotas {
	return servicequotas.New(sess, &aws.Config{
		Region: aws.String(region),
	})
}
