package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/roles"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html
//go:generate go run ../../tools/template/main.go -name instanceTypeTemplate -package main instance_type.html
//go:generate go run ../../tools/template/main.go -name keyPairTemplate -package main key_pair.html

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// Serialize the event to make it available in the browser for debugging.
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}

	var instanceType, publicKeyMaterial, terminateConfirmed string
	launched := event.QueryStringParameters["launched"]
	principalId := event.RequestContext.Authorizer["principalId"].(string)
	region := event.QueryStringParameters["region"]
	selectedRegions := strings.Split(event.StageVariables["SelectedRegions"], ",")
	terminate := event.QueryStringParameters["terminate"]
	terminated := event.QueryStringParameters["terminated"]
	if event.HTTPMethod == "POST" {
		values, err := url.ParseQuery(event.Body)
		if err != nil {
			return nil, err
		}
		instanceType = values.Get("instance_type")
		publicKeyMaterial = values.Get("public_key_material")
		region = values.Get("region")
		terminateConfirmed = values.Get("terminate")
	}

	// See if we've got a valid region or render the index page.
	found := false
	for _, r := range selectedRegions {
		found = found || r == region
	}
	if !found {
		v := struct {
			Debug                           string
			Error                           error
			Instances                       []*ec2.Instance
			Launched, Terminate, Terminated string
			Regions                         []string
		}{
			Debug:      string(b),
			Launched:   launched,
			Regions:    selectedRegions,
			Terminate:  terminate,
			Terminated: terminated,
		}
		if region != "" {
			v.Error = fmt.Errorf("%s is either not a valid region or is not in use in your organization", region)
		}
		for _, region := range selectedRegions {

			sess, err := awssessions.NewSession(awssessions.Config{Region: region})
			if err != nil {
				return nil, err
			}
			svc := ec2.New(sess)

			instances, err := awsec2.DescribeInstances(svc /* TODO filters */)
			if err != nil {
				v.Error = err
				break
			}
			v.Instances = append(v.Instances, instances...)

		}
		body, err := lambdautil.RenderHTML(indexTemplate(), v)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil
	}

	sess, err := awssessions.NewSession(awssessions.Config{Region: region})
	if err != nil {
		return nil, err
	}
	svc := ec2.New(sess)

	// We've got a region. If we're to terminate an instance, we've got enough
	// information to do so already.
	if terminateConfirmed != "" {
		if err := awsec2.TerminateInstance(svc, terminateConfirmed); err != nil {
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
		if _, err := awsec2.ImportKeyPair(svc, principalId, publicKeyMaterial); err != nil {
			return lambdautil.ErrorResponse(err)
		}
	}
	keyPairs, err := awsec2.DescribeKeyPairs(svc, principalId)
	if err != nil || len(keyPairs) != 1 {
		v := struct {
			Debug       string
			Error       error
			PrincipalId string
			Region      string
		}{
			Debug:       string(b),
			PrincipalId: principalId,
			Region:      region,
		}
		body, err := lambdautil.RenderHTML(keyPairTemplate(), v)
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
	offerings, err := awsec2.DescribeInstanceTypeOfferings(svc)
	if err != nil {
		return nil, err
	}
	instanceFamilies := make(map[string][]string)
	for _, offering := range offerings {
		instanceType := aws.StringValue(offering.InstanceType)
		ss := strings.SplitN(instanceType, ".", 2)

		// Don't bother offering some really old instance families just to save screen real estate.
		if ss[0] == "c1" || ss[0] == "cc2" || ss[0] == "c3" || ss[0] == "m1" || ss[0] == "m2" || ss[0] == "m3" || ss[0] == "r3" || ss[0] == "t1" {
			continue
		}

		instanceTypes, ok := instanceFamilies[ss[0]]
		if ok {
			instanceFamilies[ss[0]] = append(instanceTypes, instanceType)
		} else {
			instanceFamilies[ss[0]] = []string{instanceType}
		}
	}
	for _, instanceTypes := range instanceFamilies {
		sort.Slice(instanceTypes, func(i, j int) bool {
			ssI, ssJ := strings.SplitN(instanceTypes[i], ".", 2), strings.SplitN(instanceTypes[j], ".", 2)
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
	if !found {
		v := struct {
			Debug            string
			Error            error
			InstanceFamilies map[string][]string
			Region           string
		}{
			Debug:            string(b),
			InstanceFamilies: instanceFamilies,
			Region:           region,
		}
		if instanceType != "" {
			v.Error = fmt.Errorf("%s is not a valid instance type in %s", instanceType, region)
		}
		body, err := lambdautil.RenderHTML(instanceTypeTemplate(), v)
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
	image, err := awsec2.LatestAmazonLinux2AMI(svc, awsec2.X86_64)
	if err != nil {
		return nil, err
	}
	subnet, err := randomSubnet("admin", event.RequestContext.Stage, region)
	if err != nil {
		return nil, err
	}
	securityGroups, err := awsec2.DescribeSecurityGroups(svc, aws.StringValue(subnet.VpcId), "substrate-instance-factory")
	if err != nil {
		return nil, err
	}
	if len(securityGroups) != 1 {
		return nil, fmt.Errorf("security group not found in %s", aws.StringValue(subnet.VpcId))
	}
	reservation, err := awsec2.RunInstance(
		svc,
		roles.Administrator, // there's an instance profile for this role with the same name
		aws.StringValue(image.ImageId),
		instanceType,
		aws.StringValue(keyPairs[0].KeyName),
		aws.StringValue(securityGroups[0].GroupId),
		aws.StringValue(subnet.SubnetId),
		[]*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Manager"),
				Value: aws.String("substrate-instance-factory"),
			},
		},
	)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	return &events.APIGatewayProxyResponse{
		Body: fmt.Sprintf("launching %s", reservation.Instances[0].InstanceId),
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Location": location(
				event,
				url.Values{"launched": []string{aws.StringValue(reservation.Instances[0].InstanceId)}},
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

func main() {
	lambda.Start(handle)
}

func randomSubnet(environment, quality, region string) (*ec2.Subnet, error) {
	sess, err := awssessions.InSpecialAccount(accounts.Network, roles.Auditor, awssessions.Config{Region: region})
	if err != nil {
		return nil, err
	}
	svc := ec2.New(sess)
	vpcs, err := awsec2.DescribeVpcs(svc, environment, quality)
	if err != nil {
		return nil, err
	}
	if len(vpcs) != 1 {
		return nil, fmt.Errorf("%s %s VPC not found in %s", environment, quality, region)
	}
	subnets, err := awsec2.DescribeSubnets(svc, aws.StringValue(vpcs[0].VpcId))
	if err != nil {
		return nil, err
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets in %s", aws.StringValue(vpcs[0].VpcId))
	}
	return subnets[rand.Intn(len(subnets))], nil
}
