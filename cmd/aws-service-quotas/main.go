package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/pflag"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

func main() {
	global := pflag.Bool("global", false, "show or increase the service quota for a global AWS service")
	allRegions := pflag.Bool("all-regions", false, "show or increase the service quota in all AWS regions")
	region := pflag.String("region", "", "AWS region in which the service quota should be shown or increased")
	listServices := pflag.Bool("list-services", false, "list all services that have service limits to learn their -service-code values")
	listQuotas := pflag.Bool("list-quotas", false, "list all service quotes for -service-code to learn their -quota-code values")
	quotaCode := pflag.String("quota-code", "", "quota code to pass to AWS")
	serviceCode := pflag.String("service-code", "", "service code to pass to AWS")
	requiredValue := pflag.Float64("required-value", 0, "minimum required value for the service quota")
	desiredValue := pflag.Float64("desired-value", 0, "desired value for the service quota, used if the quota's current value is below -required-value")
	pflag.ErrHelp = errors.New("")
	pflag.Usage = func() {
		ui.Print("Usage: aws-service-quotas -global|-all-regions|-region <region> -list-services")
		ui.Print("       aws-service-quotas -global|-all-regions|-region <region> -service-code <code> -list-quotas")
		ui.Print("       aws-service-quotas -global|-all-regions|-region <region> -service-code <code> -quota-code <code> [-required-value <value> [-desired-value <value>]]")
		pflag.PrintDefaults()
	}
	pflag.Parse()
	if !*global && !*allRegions && *region == "" || *global && *allRegions || *global && *region != "" || *allRegions && *region != "" {
		log.Fatal("exactly one of -global, -all-regions, or a valid -region is required")
	}
	var regionSlice []string
	if *global {
		*region = "us-east-1" // Service Quotas has a hard dependency on us-east-1 for global services
	}
	if *allRegions {
		for _, region := range regions.Selected() {
			regionSlice = append(regionSlice, region)
		}
	} else {
		regionSlice = []string{*region}
	}

	ctx := context.Background()
	cfg, err := awscfg.NewConfig(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	if *listServices {
		var lines []string
		for _, region := range regionSlice {
			services, err := awsservicequotas.ListServices(ctx, cfg.Regional(region))
			if err != nil {
				ui.Fatal(err)
			}
			for _, service := range services {
				lines = append(lines, fmt.Sprintf(
					"%-23s %s\n",
					aws.ToString(service.ServiceCode),
					aws.ToString(service.ServiceName),
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
				ctx,
				cfg.Regional(region),
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
				ctx,
				cfg,
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
				ctx,
				cfg.Regional(*region),
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
				ctx,
				cfg.Regional(region),
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
