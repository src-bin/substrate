package awsroute53

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func FindHostedZone(svc *route53.Route53, name string) (*route53.HostedZone, error) {
	zones, err := ListHostedZones(svc)
	if err != nil {
		return nil, err
	}
	for _, zone := range zones {
		if aws.StringValue(zone.Name) == name {
			return zone, nil
		}
	}
	return nil, HostedZoneNotFoundError(name)
}

type HostedZoneNotFoundError string

func (err HostedZoneNotFoundError) Error() string {
	return fmt.Sprintf("HostedZoneNotFoundError: %s not found", string(err))
}

func ListHostedZones(svc *route53.Route53) (zones []*route53.HostedZone, err error) {
	var marker *string
	for {
		in := &route53.ListHostedZonesInput{Marker: marker}
		out, err := svc.ListHostedZones(in)
		if err != nil {
			return nil, err
		}
		zones = append(zones, out.HostedZones...)
		if marker = out.NextMarker; marker == nil {
			break
		}
	}
	return
}
