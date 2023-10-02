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
