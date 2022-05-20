package awsiam

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
)

const SubstrateManaged = "SubstrateManaged"

// NOT DONE!
func EnsurePolicy(ctx context.Context, cfg *awscfg.Config, name, content string) error {
	policy, err := createPolicy(ctx, cfg, name, content)
	/*
		if awsutil.ErrorCodeIs(err, DuplicatePolicyException) {
			err = nil

			for policySummary := range ListPolicies(ctx, cfg, policyType) {
				if aws.ToString(policySummary.Name) != name {
					continue
				}

				policy, err = updatePolicy(ctx, cfg, aws.ToString(policySummary.Id), name, content)
				if err != nil {
					return err
				}
			}

		}
		if err != nil {
			return err
		}
	*/
	log.Printf("%+v", policy)

	return err
}

func attachUserPolicy(ctx context.Context, cfg *awscfg.Config, username, policyArn string) error {
	in := &iam.AttachUserPolicyInput{
		PolicyArn: aws.String(policyArn),
		UserName:  aws.String(username),
	}
	_, err := cfg.IAM().AttachUserPolicy(ctx, in)
	/*
		if awsutil.ErrorCodeIs(err, DuplicatePolicyAttachmentException) {
			err = nil
		}
	*/
	return err
}

func createPolicy(ctx context.Context, cfg *awscfg.Config, name, content string) (*types.Policy, error) {
	in := &iam.CreatePolicyInput{
		PolicyDocument: aws.String(content),
		PolicyName:     aws.String(name),
	}
	out, err := cfg.IAM().CreatePolicy(ctx, in)
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}
