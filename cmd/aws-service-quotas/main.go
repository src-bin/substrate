package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/version"
)

func main() {

	accountId := flag.String("account-number", "", "AWS account number")
	roleName := flag.String("role", "", "AWS IAM role name to assume")
	region := flag.String("region", "", "AWS region in which the service quota should be shown or increased")
	listServices := flag.Bool("list-services", false, "list all services that have service limits to learn their -service-code values")
	listQuotas := flag.Bool("list-quotas", false, "list all service quotes for -service-code to learn their -quota-code values")
	allRegions := flag.Bool("all-regions", false, "show or increase the service quota in all AWS regions")
	quotaCode := flag.String("quota-code", "", "quota code to pass to AWS")
	serviceCode := flag.String("service-code", "", "service code to pass to AWS")
	requiredValue := flag.Float64("required-value", 0, "minimum required value for the service quota")
	desiredValue := flag.Float64("desired-value", 0, "desired value for the service quota, used if the quota's current value is below -required-value")
	flag.Parse()
	version.Flag()
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
		awssessions.Must(awssessions.NewSession(awssessions.Config{})),
		*accountId,
		*roleName,
	)

	if *listServices {
		var lines []string
		for _, region := range regionSlice {
			services, err := awsservicequotas.ListServices(servicequotasNew(sess, region))
			if err != nil {
				log.Fatal(err)
			}
			for _, service := range services {
				lines = append(lines, fmt.Sprintf(
					"%-23s %s\n",
					aws.StringValue(service.ServiceCode),
					aws.StringValue(service.ServiceName),
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
			quotas, err := awsservicequotas.ListServiceQuotas(
				servicequotasNew(sess, region),
				*serviceCode,
			)
			if err != nil {
				log.Fatal(err)
			}
			for _, quota := range quotas {
				fmt.Printf("%+v\n", quota)
			}
		}
		return
	}

	if *quotaCode == "" || *serviceCode == "" {
		log.Fatal("-quota-code and -service-code are required without -list-services or -list-quotas")
	}
	if *requiredValue > 0 {
		if *allRegions {

			if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
				sess,
				*quotaCode,
				*serviceCode,
				*requiredValue,
				*desiredValue,
				time.Time{},
			); err != nil {
				log.Fatal(err)
			}

		} else {

			if err := awsservicequotas.EnsureServiceQuota(
				servicequotasNew(sess, *region),
				*quotaCode,
				*serviceCode,
				*requiredValue,
				*desiredValue,
				time.Time{},
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
