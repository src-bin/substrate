package instancefactory

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
)

const InstanceFactory = "InstanceFactory"

func Main2(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {

	var (
		instanceType                          awsec2.InstanceType
		publicKeyMaterial, terminateConfirmed string
	)
	launched := event.QueryStringParameters["launched"] // TODO don't propagate this into the HTML if the instance it references is in the "running" state
	principalId := event.RequestContext.Authorizer.Lambda[authorizerutil.PrincipalId].(string)
	region := event.QueryStringParameters["region"]
	//log.Printf("GET region: %+v", region)
	selectedRegions := strings.Split(os.Getenv("SELECTED_REGIONS"), ",")
	//log.Printf("selectedRegions: %+v", selectedRegions)
	terminate := event.QueryStringParameters["terminate"]
	terminated := event.QueryStringParameters["terminated"]
	if event.RequestContext.HTTP.Method == "POST" {
		body, err := lambdautil.EventBody2(event)
		if err != nil {
			return lambdautil.ErrorResponse2(err)
		}
		values, err := url.ParseQuery(body)
		if err != nil {
			return lambdautil.ErrorResponse2(err)
		}
		if err := lambdautil.PreventCSRF2(values, event); err != nil {
			return lambdautil.ErrorResponse2(err)
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
			CSRF                            string
			Error                           error
			Instances                       []awsec2.Instance
			Launched, Terminate, Terminated string
			Regions                         []string
		}{
			CSRF:       lambdautil.CSRFCookie2(event),
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
						Name:   aws.String(fmt.Sprintf("tag:%s", tagging.Manager)),
						Values: []string{tagging.Substrate},
					},
					{
						Name:   aws.String("key-name"),
						Values: []string{fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.PrincipalId])},
					},
				},
			)
			if err != nil {
				v.Error = err
				break
			}
			v.Instances = append(v.Instances, instances...)
		}
		body, err := lambdautil.RenderHTML(html, v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayV2HTTPResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
			StatusCode: http.StatusOK,
		}, nil
	}

	cfg = cfg.Regional(region)

	// We've got a region. If we're to terminate an instance, we've got enough
	// information to do so already.
	if terminateConfirmed != "" {
		if err := awsec2.TerminateInstance(ctx, cfg, terminateConfirmed); err != nil {
			return lambdautil.ErrorResponse2(err)
		}
		return &events.APIGatewayV2HTTPResponse{
			Body: fmt.Sprintf("terminating %s", terminateConfirmed),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Location": lambdautil.Location(
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
			return lambdautil.ErrorResponse2(err)
		}
	}
	keyPairs, err := awsec2.DescribeKeyPairs(ctx, cfg, principalId)
	if err != nil || len(keyPairs) != 1 {
		v := struct {
			CSRF        string
			Error       error
			PrincipalId string
			Region      string
		}{
			CSRF:        lambdautil.CSRFCookie2(event),
			PrincipalId: principalId,
			Region:      region,
		}
		body, err := lambdautil.RenderHTML(htmlForKeyPair, v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayV2HTTPResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
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
			CSRF             string
			Error            error
			InstanceFamilies map[string][]awsec2.InstanceType
			Region           string
		}{
			CSRF:             lambdautil.CSRFCookie2(event),
			InstanceFamilies: instanceFamilies,
			Region:           region,
		}
		if instanceType != "" {
			v.Error = fmt.Errorf("%s is not a valid instance type in %s", instanceType, region)
		}
		body, err := lambdautil.RenderHTML(htmlForType, v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayV2HTTPResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
			StatusCode: http.StatusOK,
		}, nil
	}

	// Let's do this! Start by figuring out whether to provide an AMI and, if
	// so, which one (the latest Amazon Linux 2 AMI, of course).
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
	launchTemplateName := fmt.Sprintf("%s-%s", InstanceFactory, arch)
	launchTemplate, err := awsec2.DescribeLaunchTemplateVersion(ctx, cfg, launchTemplateName)
	if err != nil {
		if awsutil.ErrorCodeIs(err, awsec2.InvalidLaunchTemplateName_NotFoundException) {
			launchTemplateName = ""
		} else {
			return nil, err
		}
	}
	var imageId string
	if launchTemplate != nil && launchTemplate.LaunchTemplateData.ImageId != nil {
		imageId = aws.ToString(launchTemplate.LaunchTemplateData.ImageId)
	} else {
		image, err := awsec2.LatestAmazonLinux2AMI(ctx, cfg, arch)
		if err != nil {
			return nil, err
		}
		imageId = aws.ToString(image.ImageId)
	}

	// Next decide where to situate the instance in the network.
	substrateAccount, err := cfg.FindSubstrateAccount(ctx)
	if err != nil {
		return nil, err
	}
	quality := substrateAccount.Tags[tagging.Quality]
	if quality == "" {
		quality = naming.Default
	}
	subnet, err := randomSubnet(ctx, cfg, accounts.Admin, quality, region)
	//subnet, err := randomSubnet(ctx, cfg, accounts.Admin, event.RequestContext.Stage, region)
	if err != nil {
		return nil, err
	}
	securityGroups, err := awsec2.DescribeSecurityGroups(ctx, cfg, aws.ToString(subnet.VpcId), InstanceFactory)
	if err != nil {
		return nil, err
	}
	if len(securityGroups) != 1 {
		return nil, fmt.Errorf("security group not found in %s", aws.ToString(subnet.VpcId))
	}

	// Provision the instance! Tell the caller all about it.
	reservation, err := awsec2.RunInstance(
		ctx,
		cfg,
		event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName].(string),
		imageId,
		instanceType,
		aws.ToString(keyPairs[0].KeyName),
		launchTemplateName,
		100, // gigabyte root volume
		aws.ToString(securityGroups[0].GroupId),
		aws.ToString(subnet.SubnetId),
		[]awsec2.Tag{{
			Key:   aws.String(tagging.Manager),
			Value: aws.String(tagging.Substrate),
		}},
	)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}
	return &events.APIGatewayV2HTTPResponse{
		Body: fmt.Sprintf("launching %s", aws.ToString(reservation.Instances[0].InstanceId)),
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Location": lambdautil.Location(
				event,
				url.Values{"launched": []string{aws.ToString(reservation.Instances[0].InstanceId)}},
			),
		},
		StatusCode: http.StatusFound,
	}, nil

}

func randomSubnet(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality, region string,
) (*awsec2.Subnet, error) {
	cfg, err := cfg.AssumeSpecialRole(ctx, accounts.Network, roles.Auditor, time.Hour)
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

//go:embed instance-factory.html
var html string

//go:embed key-pair.html
var htmlForKeyPair string

//go:embed type.html
var htmlForType string
