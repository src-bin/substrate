package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/tagging"
)

const AccessDeniedException = "AccessDeniedException"

type Instance struct {
	types.InstanceMetadata
	Region string
	Tags   tagging.Map
}

func ListInstances(ctx context.Context, mgmtCfg *awscfg.Config) (instances []*Instance, err error) {
	for _, region := range regions.Selected() {
		client := mgmtCfg.Regional(region).SSOAdmin()
		var nextToken *string
		for {
			var out *ssoadmin.ListInstancesOutput
			out, err = client.ListInstances(ctx, &ssoadmin.ListInstancesInput{
				NextToken: nextToken,
			})
			if awsutil.ErrorCodeIs(err, AccessDeniedException) { // hostile lie of a response instead of just [] (or not but we can't tell)
				err = nil
				break
			} else if err != nil {
				return
			}
			for _, im := range out.Instances {
				instance := &Instance{
					InstanceMetadata: im,
					Region:           region,
					Tags:             tagging.Map{},
				}

				out, err := client.ListTagsForResource(ctx, &ssoadmin.ListTagsForResourceInput{
					InstanceArn: im.InstanceArn,
					ResourceArn: im.InstanceArn,
				})
				if err != nil {
					return nil, err
				}
				for _, tag := range out.Tags {
					instance.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
				}

				instances = append(instances, instance)
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
	}
	return
}

func TagInstance(ctx context.Context, mgmtCfg *awscfg.Config, instance *Instance, tags tagging.Map) error {
	_, err := mgmtCfg.Regional(instance.Region).SSOAdmin().TagResource(ctx, &ssoadmin.TagResourceInput{
		InstanceArn: instance.InstanceArn,
		ResourceArn: instance.InstanceArn,
		Tags:        tagStructs(tags),
	})
	return err
}

func tagStructs(tags tagging.Map) []types.Tag {
	structs := make([]types.Tag, 0, len(tags))
	for key, value := range tags {
		structs = append(structs, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return structs
}
