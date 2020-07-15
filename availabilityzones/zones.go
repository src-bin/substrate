package availabilityzones

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/src-bin/substrate/awsec2"
)

const NumberPerNetwork = 3

// Select returns a list of up to n availability zone names in the given
// region, chosen in order from newest to oldest but returned in lexically
// sorted order.
func Select(sess *session.Session, region string, n int) ([]string, error) {

	zones, err := awsec2.DescribeAvailabilityZones(
		ec2.New(sess, &aws.Config{Region: aws.String(region)}),
		region,
	)
	if err != nil {
		return nil, err
	}

	s := make(zoneIdNameSlice, len(zones))
	for i, az := range zones {
		s[i] = zoneIdName{aws.StringValue(az.ZoneId), aws.StringValue(az.ZoneName)}
	}
	sort.Sort(s)
	names := make([]string, 0, n)
	for i := len(s) - 1; i >= 0 && len(s)-i <= n; i-- {
		names = append(names, s[i].Name)
	}
	sort.Strings(names)

	return names, nil
}

type zoneIdName struct {
	Id, Name string
}

type zoneIdNameSlice []zoneIdName

func (s zoneIdNameSlice) Len() int           { return len(s) }
func (s zoneIdNameSlice) Less(i, j int) bool { return s[i].Id < s[j].Id }
func (s zoneIdNameSlice) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}
