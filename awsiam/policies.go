package awsiam

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam/awsiamusers"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
)

const (
	DuplicatePolicyException = "DuplicatePolicyException"
	SubstrateManaged         = awsiamusers.SubstrateManaged
)

type (
	Policy        = types.Policy
	PolicyVersion = types.PolicyVersion
)

func CreatePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	doc *policies.Document,
) (*Policy, error) {
	in := &iam.CreatePolicyInput{
		PolicyDocument: aws.String(jsonutil.MustString(doc)),
		PolicyName:     aws.String(name),
		// TODO Tags
	}
	out, err := cfg.IAM().CreatePolicy(ctx, in)
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}

func CreatePolicyVersion(
	ctx context.Context,
	cfg *awscfg.Config,
	arn string,
	doc *policies.Document,
) (*Policy, error) {
	in := &iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(arn),
		PolicyDocument: aws.String(jsonutil.MustString(doc)),
		SetAsDefault:   true,
	}
	_, err := cfg.IAM().CreatePolicyVersion(ctx, in)
	if err != nil {
		return nil, err
	}
	out, err := cfg.IAM().GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}

func EnsurePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	doc *policies.Document,
) (*Policy, error) {
	ui.Spinf("creating the %s IAM policy", name)
	policy, err := CreatePolicy(ctx, cfg, name, doc)
	if awsutil.ErrorCodeIs(err, DuplicatePolicyException) || awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		ui.Stop("already exists")
		ui.Spinf("updating the %s IAM policy", name)
		var policies []Policy
		policies, err = ListPolicies(ctx, cfg)
		if err != nil {
			return nil, ui.StopErr(err)
		}
		for _, p := range policies {
			arn := aws.ToString(p.Arn)
			if aws.ToString(p.PolicyName) == name {
				var policyVersions []PolicyVersion
				if policyVersions, err = listPolicyVersions(ctx, cfg, arn); err != nil {
					return nil, ui.StopErr(err)
				}

				// TODO compare doc to the default version and short-circuit if they're the same

				// Proactively delete the oldest policy version if we're at
				// the limit so we can create the new version.
				if len(policyVersions) >= 5 { // 5 is the hard limit (as of now)
					sort.Slice(policyVersions, func(i, j int) bool {
						return aws.ToTime(policyVersions[i].CreateDate).Compare(aws.ToTime(policyVersions[j].CreateDate)) < 0
					})
					if err = deletePolicyVersion(ctx, cfg, arn, aws.ToString(policyVersions[0].VersionId)); err != nil {
						return nil, ui.StopErr(err)
					}
				}

				policy, err = CreatePolicyVersion(ctx, cfg, arn, doc)
				break
			}
		}
	}
	//log.Print(jsonutil.MustString(policy))
	return policy, ui.StopErr(err)
}

func ListPolicies(ctx context.Context, cfg *awscfg.Config) ([]Policy, error) {
	var (
		policies []Policy
		marker   *string
	)
	for {
		out, err := cfg.IAM().ListPolicies(ctx, &iam.ListPoliciesInput{
			Marker: marker,
			Scope:  types.PolicyScopeTypeLocal,
		})
		if err != nil {
			return nil, err
		}
		for _, policy := range out.Policies {
			policies = append(policies, policy)
		}
		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}
	//log.Print(jsonutil.MustString(policies))
	return policies, nil
}

func deletePolicyVersion(ctx context.Context, cfg *awscfg.Config, arn, version string) error {
	in := &iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(arn),
		VersionId: aws.String(version),
	}
	_, err := cfg.IAM().DeletePolicyVersion(ctx, in)
	return err
}

func listPolicyVersions(ctx context.Context, cfg *awscfg.Config, arn string) ([]PolicyVersion, error) {
	var (
		policyVersions []PolicyVersion
		marker         *string
	)
	for {
		out, err := cfg.IAM().ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{
			Marker:    marker,
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return nil, err
		}
		for _, policyVersion := range out.Versions {
			policyVersions = append(policyVersions, policyVersion)
		}
		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}
	//log.Print(jsonutil.MustString(policyVersions))
	return policyVersions, nil
}
