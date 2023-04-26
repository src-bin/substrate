package awsiam

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/ui"
)

type InstanceProfile = types.InstanceProfile

func CreateInstanceProfile(ctx context.Context, cfg *awscfg.Config, roleName string) (*InstanceProfile, error) {
	client := cfg.IAM()
	out, err := client.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(roleName),
		Tags:                tagsFor(roleName),
	})
	if err != nil {
		return nil, err
	}
	if _, err := client.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(roleName),
		RoleName:            aws.String(roleName),
	}); err != nil {
		return nil, err
	}
	return out.InstanceProfile, nil
}

func DeleteInstanceProfile(ctx context.Context, cfg *awscfg.Config, roleName string) (err error) {
	client := cfg.IAM()
	if _, err = client.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: aws.String(roleName),
		RoleName:            aws.String(roleName),
	}); err != nil {
		return
	}
	for i := 0; i < 10; i++ {
		_, err = client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
			InstanceProfileName: aws.String(roleName),
		})
		if !awsutil.ErrorCodeIs(err, DeleteConflict) {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	return
}

func EnsureInstanceProfile(ctx context.Context, cfg *awscfg.Config, roleName string) (instProf *InstanceProfile, err error) {
	ui.Spinf("creating an EC2 instance profile for %s", roleName)
	instProf, err = CreateInstanceProfile(ctx, cfg, roleName)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		ui.Stop("already exists")
		err = nil
		return
	}
	if awsutil.ErrorCodeIs(err, LimitExceeded) {
		err = nil // there's an outside chance that masking this masks an instance profile with the wrong role
	}
	ui.StopErr(err)
	return
}
