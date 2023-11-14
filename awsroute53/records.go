package awsroute53

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/src-bin/substrate/awscfg"
)

const (
	CREATE = types.ChangeActionCreate
	DELETE = types.ChangeActionDelete
	UPSERT = types.ChangeActionUpsert

	A     = types.RRTypeA
	AAAA  = types.RRTypeAaaa
	CNAME = types.RRTypeCname
	TXT   = types.RRTypeTxt
)

type (
	AliasTarget             = types.AliasTarget
	Change                  = types.Change
	ChangeAction            = types.ChangeAction
	RRType                  = types.RRType
	ResourceRecord          = types.ResourceRecord
	ResourceRecordSet       = types.ResourceRecordSet
	ResourceRecordSetRegion = types.ResourceRecordSetRegion
)

func ChangeResourceRecordSets(
	ctx context.Context,
	cfg *awscfg.Config,
	zoneId string,
	changes []Change,
) error {
	_, err := cfg.Route53().ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
		HostedZoneId: aws.String(zoneId),
	})
	return err
}

// DeleteResourceRecordSets deletes one or more names from the given zone
// regardless of what responses are configured for those names.
func DeleteResourceRecordSets(
	ctx context.Context,
	cfg *awscfg.Config,
	zoneId string,
	f func(ResourceRecordSet) bool,
) error {
	records, err := ListResourceRecordSets(ctx, cfg, zoneId)
	if err != nil {
		return err
	}
	var changes []Change
	for _, record := range records {
		if f(record) {
			var recordPtr *ResourceRecordSet
			*recordPtr = record
			changes = append(
				changes,
				Change{
					Action:            DELETE,
					ResourceRecordSet: recordPtr,
				},
			)
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return ChangeResourceRecordSets(ctx, cfg, zoneId, changes)
}

func ListResourceRecordSets(
	ctx context.Context,
	cfg *awscfg.Config,
	zoneId string,
) (records []ResourceRecordSet, err error) {
	client := cfg.Route53()
	var (
		nextRecordIdentifier, nextRecordName *string
		nextRecordType                       types.RRType
	)
	for {
		var out *route53.ListResourceRecordSetsOutput
		if out, err = client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
			HostedZoneId:          aws.String(zoneId),
			StartRecordIdentifier: nextRecordIdentifier,
			StartRecordName:       nextRecordName,
			StartRecordType:       nextRecordType,
		}); err != nil {
			return
		}
		records = append(records, out.ResourceRecordSets...)
		if !out.IsTruncated {
			break
		}
		nextRecordIdentifier = out.NextRecordIdentifier
		nextRecordName = out.NextRecordName
		nextRecordType = out.NextRecordType
	}
	return
}
