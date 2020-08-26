package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html
//go:generate go run ../../tools/template/main.go -name instanceTypeTemplate -package main instance_type.html
//go:generate go run ../../tools/template/main.go -name instanceTemplate -package main instance.html

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// Serialize the event to make it available in the browser for debugging.
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}

	var instanceType string
	region := event.QueryStringParameters["region"]
	selectedRegions := strings.Split(event.StageVariables["SelectedRegions"], ",")
	if event.HTTPMethod == "POST" {
		values, err := url.ParseQuery(event.Body)
		if err != nil {
			return nil, err
		}
		instanceType = values.Get("instance_type")
		region = values.Get("region")

	}

	// See if we've got a valid region or render the index page.
	found := false
	for _, r := range selectedRegions {
		found = found || r == region
	}
	if !found {
		v := struct {
			Debug   string
			Error   error
			Regions []string
		}{
			Debug:   string(b),
			Regions: selectedRegions,
		}
		if region != "" {
			v.Error = fmt.Errorf("%s is either not a valid region or is not in use in your organization", region)
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

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := ec2.New(
		sess,
		&aws.Config{Region: aws.String(region)},
	)

	// We've got a region. Use it to enumerate all valid instance types.
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

	// See if we've got a valid instance type or render the instance type page.
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
			v.Error = fmt.Errorf("%s is either not a valid instance type in %s", instanceType, region)
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
	v := struct {
		Debug        string
		Error        error
		InstanceType string
		Region       string
	}{
		Debug:        string(b),
		InstanceType: instanceType,
		Region:       region,
	}
	imageId, err := awsec2.LatestAmazonLinux2AMI(svc)
	if err != nil {
		v.Error = err
	}
	v.Error = fmt.Errorf("%+v", out)
	/*
		_, err := awsec2.RunInstances(svc)
		if err != nil {
			v.Error = err
		}
	*/
	body, err := lambdautil.RenderHTML(instanceTemplate(), v)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil

}

func main() {
	lambda.Start(handle)
}
