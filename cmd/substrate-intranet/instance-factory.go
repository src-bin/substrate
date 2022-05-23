package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
)

//go:generate go run ../../tools/template/main.go -name instanceFactoryTemplate -package main instance-factory.html
//go:generate go run ../../tools/template/main.go -name instanceFactoryTypeTemplate -package main instance-factory-type.html
//go:generate go run ../../tools/template/main.go -name instanceFactoryKeyPairTemplate -package main instance-factory-key-pair.html

func init() {
	handlers["/instance-factory"] = instanceFactoryHandler
}

func instanceFactoryHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	var (
		instanceType                          awsec2.InstanceType
		publicKeyMaterial, terminateConfirmed string
	)
	launched := event.QueryStringParameters["launched"] // TODO don't propagate this into the HTML if the instance it references is in the "running" state
	principalId := event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string)
	region := event.QueryStringParameters["region"]
	//log.Printf("GET region: %+v", region)
	selectedRegions := strings.Split(event.StageVariables["SelectedRegions"], ",")
	//log.Printf("selectedRegions: %+v", selectedRegions)
	terminate := event.QueryStringParameters["terminate"]
	terminated := event.QueryStringParameters["terminated"]
	if event.HTTPMethod == "POST" {
		body, err := lambdautil.EventBody(event)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}
		values, err := url.ParseQuery(body)
		if err != nil {
			return nil, err
		}
		//log.Printf("POST values: %+v", values)
		instanceType = awsec2.InstanceType(values.Get("instance_type"))
		//log.Printf("POST instanceType: %+v", instanceType)
		publicKeyMaterial = values.Get("public_key_material")
		region = values.Get("region")
		//log.Printf("POST region: %+v", region)
		terminateConfirmed = values.Get("terminate")
	}

	// See if we've got a valid region or render the index page.
	found := false
	for _, r := range selectedRegions {
		found = found || r == region
	}
	//log.Printf("found: %v", found)
	if !found {
		v := struct {
			Error                           error
			Instances                       []awsec2.Instance
			Launched, Terminate, Terminated string
			Regions                         []string
		}{
			Launched:   launched,
			Regions:    selectedRegions,
			Terminate:  terminate,
			Terminated: terminated,
		}
		if region != "" {
			v.Error = fmt.Errorf("%s is either not a valid region or is not in use in your organization", region)
		}
		for _, region := range selectedRegions {
			instances, err := awsec2.DescribeInstances(
				ctx,
				cfg.Regional(region),
				[]awsec2.Filter{
					{
						Name: aws.String(fmt.Sprintf("tag:%s", tags.Manager)),
						Values: []string{
							tags.Substrate,
							tags.SubstrateInstanceFactory, // remove in 2022.10
						},
					},
					{
						Name:   aws.String("key-name"),
						Values: []string{fmt.Sprint(event.RequestContext.Authorizer["principalId"])},
					},
				},
			)
			if err != nil {
				v.Error = err
				break
			}
			v.Instances = append(v.Instances, instances...)
		}
		body, err := lambdautil.RenderHTML(instanceFactoryTemplate(), v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil
	}

	cfg = cfg.Regional(region)

	// We've got a region. If we're to terminate an instance, we've got enough
	// information to do so already.
	if terminateConfirmed != "" {
		if err := awsec2.TerminateInstance(ctx, cfg, terminateConfirmed); err != nil {
			return lambdautil.ErrorResponse(err)
		}
		return &events.APIGatewayProxyResponse{
			Body: fmt.Sprintf("terminating %s", terminateConfirmed),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Location": location(
					event,
					url.Values{"terminated": []string{terminateConfirmed}},
				),
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	// We've got a region. Ensure we've got a key pair in this region, too, or
	// render the public key input page.
	if publicKeyMaterial != "" {
		if _, err := awsec2.ImportKeyPair(ctx, cfg, principalId, publicKeyMaterial); err != nil {
			return lambdautil.ErrorResponse(err)
		}
	}
	keyPairs, err := awsec2.DescribeKeyPairs(ctx, cfg, principalId)
	if err != nil || len(keyPairs) != 1 {
		v := struct {
			Error       error
			PrincipalId string
			Region      string
		}{
			PrincipalId: principalId,
			Region:      region,
		}
		body, err := lambdautil.RenderHTML(instanceFactoryKeyPairTemplate(), v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil
	}

	// We've got a region and a key pair. Use the region to enumerate all valid
	// instance types.
	offerings, err := awsec2.DescribeInstanceTypeOfferings(ctx, cfg)
	if err != nil {
		return nil, err
	}
	instanceFamilies := make(map[string][]awsec2.InstanceType)
	for _, offering := range offerings {
		ss := strings.SplitN(string(offering.InstanceType), ".", 2)

		// Don't bother offering some really old instance families just to save screen real estate.
		if ss[0] == "c1" || ss[0] == "cc2" || ss[0] == "c3" || ss[0] == "m1" || ss[0] == "m2" || ss[0] == "m3" || ss[0] == "r3" || ss[0] == "t1" {
			continue
		}

		instanceTypes, ok := instanceFamilies[ss[0]]
		if ok {
			instanceFamilies[ss[0]] = append(instanceTypes, offering.InstanceType)
		} else {
			instanceFamilies[ss[0]] = []awsec2.InstanceType{offering.InstanceType}
		}
	}
	for _, instanceTypes := range instanceFamilies {
		sort.Slice(instanceTypes, func(i, j int) bool {
			ssI, ssJ := strings.SplitN(string(instanceTypes[i]), ".", 2), strings.SplitN(string(instanceTypes[j]), ".", 2)
			if ssI[0] < ssJ[0] {
				return true
			}
			if ssI[0] > ssJ[0] {
				return false
			}
			m := map[string]int{
				"nano":     1,
				"micro":    2,
				"small":    4,
				"medium":   8,
				"large":    16,
				"xlarge":   32,
				"2xlarge":  64,
				"3xlarge":  96,
				"4xlarge":  128,
				"6xlarge":  192,
				"8xlarge":  256,
				"9xlarge":  288,
				"10xlarge": 320,
				"12xlarge": 384,
				"16xlarge": 512,
				"18xlarge": 576,
				"24xlarge": 768,
				"32xlarge": 1024,
				"metal":    2048,
			}
			return m[ssI[1]] < m[ssJ[1]]
		})
	}

	// See if they've selected a valid instance type. If not, render the
	// instance type selection page.
	found = false
	for _, instanceTypes := range instanceFamilies {
		for _, i := range instanceTypes {
			found = found || i == instanceType
		}
	}
	//log.Printf("found: %v", found)
	if !found {
		v := struct {
			Error            error
			InstanceFamilies map[string][]awsec2.InstanceType
			Region           string
		}{
			InstanceFamilies: instanceFamilies,
			Region:           region,
		}
		if instanceType != "" {
			v.Error = fmt.Errorf("%s is not a valid instance type in %s", instanceType, region)
		}
		body, err := lambdautil.RenderHTML(instanceFactoryTypeTemplate(), v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil
	}

	// Let's do this!
	instanceTypes, err := awsec2.DescribeInstanceTypes(ctx, cfg, []awsec2.InstanceType{instanceType})
	if err != nil {
		return nil, err
	}
	if len(instanceTypes) == 0 {
		return nil, fmt.Errorf("instance type %s not found", instanceType)
	}
	if len(instanceTypes) > 1 {
		return nil, fmt.Errorf("%d instance types %s found", len(instanceTypes), instanceType)
	}
	archs := instanceTypes[0].ProcessorInfo.SupportedArchitectures
	if len(archs) == 0 {
		return nil, fmt.Errorf("instance type %s supports zero CPU architectures", instanceType)
	}
	if len(archs) > 2 {
		return nil, fmt.Errorf("instance type %s supports more than two CPU architectures", instanceType)
	}
	arch := archs[0]
	if arch == "i386" && len(archs) == 2 {
		arch = archs[1]
	}
	image, err := awsec2.LatestAmazonLinux2AMI(ctx, cfg, arch)
	if err != nil {
		return nil, err
	}

	subnet, err := randomSubnet(ctx, cfg, accounts.Admin, event.RequestContext.Stage, region)
	if err != nil {
		return nil, err
	}
	securityGroups, err := awsec2.DescribeSecurityGroups(ctx, cfg, aws.ToString(subnet.VpcId), "InstanceFactory")
	if err != nil {
		return nil, err
	}
	if len(securityGroups) != 1 {
		return nil, fmt.Errorf("security group not found in %s", aws.ToString(subnet.VpcId))
	}

	reservation, err := awsec2.RunInstance(
		ctx,
		cfg,
		event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
		aws.ToString(image.ImageId),
		instanceType,
		aws.ToString(keyPairs[0].KeyName),
		"InstanceFactory",
		100, // gigabyte root volume
		aws.ToString(securityGroups[0].GroupId),
		aws.ToString(subnet.SubnetId),
		[]awsec2.Tag{{
			Key:   aws.String(tags.Manager),
			Value: aws.String(tags.Substrate),
		}},
	)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}

	return &events.APIGatewayProxyResponse{
		Body: fmt.Sprintf("launching %s", aws.ToString(reservation.Instances[0].InstanceId)),
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Location": location(
				event,
				url.Values{"launched": []string{aws.ToString(reservation.Instances[0].InstanceId)}},
			),
		},
		StatusCode: http.StatusFound,
	}, nil

}

func location(event *events.APIGatewayProxyRequest, query url.Values) string {
	u := &url.URL{
		Scheme:   "https",
		Host:     event.Headers["Host"],
		Path:     event.Path,
		RawQuery: query.Encode(),
	}
	return u.String()
}

func randomSubnet(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality, region string,
) (*awsec2.Subnet, error) {
	cfg, err := cfg.AssumeSpecialRole(ctx, accounts.Network, roles.Auditor, "", time.Hour)
	if err != nil {
		return nil, err
	}
	// TODO cfg region
	vpcs, err := awsec2.DescribeVpcs(ctx, cfg, environment, quality)
	if err != nil {
		return nil, err
	}
	if len(vpcs) != 1 {
		return nil, fmt.Errorf("%s %s VPC not found in %s", environment, quality, region)
	}
	subnets, err := awsec2.DescribeSubnets(ctx, cfg, aws.ToString(vpcs[0].VpcId))
	if err != nil {
		return nil, err
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets in %s", aws.ToString(vpcs[0].VpcId))
	}
	subnet := subnets[rand.Intn(len(subnets))] // don't leak the slice
	return &subnet, nil
}
