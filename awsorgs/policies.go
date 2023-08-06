package awsorgs

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

type (
	Policy        = types.Policy
	PolicySummary = types.PolicySummary
	PolicyType    = types.PolicyType
)

const (
	ConcurrentModificationException               = "ConcurrentModificationException"
	DuplicatePolicyAttachmentException            = "DuplicatePolicyAttachmentException"
	DuplicatePolicyException                      = "DuplicatePolicyException"
	PolicyTypeAlreadyEnabledException             = "PolicyTypeAlreadyEnabledException"
	SERVICE_CONTROL_POLICY             PolicyType = "SERVICE_CONTROL_POLICY"
	TAG_POLICY                         PolicyType = "TAG_POLICY"
)

func DescribePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	policyId string,
) (*Policy, error) {
	out, err := cfg.Organizations().DescribePolicy(ctx, &organizations.DescribePolicyInput{
		PolicyId: aws.String(policyId),
	})
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}

func EnablePolicyType(
	ctx context.Context,
	cfg *awscfg.Config,
	policyType PolicyType,
) error {
	time.Sleep(5e9)       // halfhearted attempt to avoid "ConcurrentModificationException: AWS Organizations can't complete your request because it conflicts with another attempt to modify the same entity. Try again later."
	defer time.Sleep(5e9) // should be more graceful but one last sleep should prevent ConcurrentModificationException
	for {
		root, err := DescribeRoot(ctx, cfg)
		if err != nil {
			log.Fatal(err)
		}
		for _, pt := range root.PolicyTypes {
			if pt.Status == types.PolicyTypeStatusEnabled && pt.Type == policyType {
				return nil
			}
		}
		if _, err := cfg.Organizations().EnablePolicyType(ctx, &organizations.EnablePolicyTypeInput{
			PolicyType: policyType,
			RootId:     root.Id,
		}); err != nil && !awsutil.ErrorCodeIs(err, ConcurrentModificationException) {
			return err
		}
		time.Sleep(5e9) // TODO exponential backoff
	}
}

// EnsurePolicy makes potentially several AWS API requests to ensure,
// regardless of initial state, that a policy by a given name and type exists,
// has the desired content, and is attached to the specified root.
//
// A curiosity:  Though DescribeOrganization alludes to the ability to attach
// service control policies to the management account, that does not appear to be
// possible (without first enabling service control policies).  It's unclear
// if such policies would apply only to that account if attached there instead
// of the root.  In other words, the distinction between a management account and
// the root is murkier than I thought yesterday.
func EnsurePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	root *Root,
	name string,
	policyType PolicyType,
	doc *policies.Document,
) error {

	err := enablePolicyType(ctx, cfg, aws.ToString(root.Id), policyType)
	if awsutil.ErrorCodeIs(err, PolicyTypeAlreadyEnabledException) {
		err = nil
	}
	if err != nil {
		return err
	}

	policy, err := createPolicy(ctx, cfg, name, policyType, doc)
	if awsutil.ErrorCodeIs(err, DuplicatePolicyException) {
		err = nil

		summaries, err := ListPolicies(ctx, cfg, policyType)
		if err != nil {
			return err
		}
		for _, summary := range summaries {
			if aws.ToString(summary.Name) != name {
				continue
			}

			policy, err = updatePolicy(ctx, cfg, aws.ToString(summary.Id), name, doc)
			if err != nil {
				return err
			}
		}

	}
	if err != nil {
		return err
	}
	//log.Printf("%+v", policy)

	// TODO tag the policy, which isn't a supported operation; get clever

	err = attachPolicy(ctx, cfg, aws.ToString(policy.PolicySummary.Id), aws.ToString(root.Id))
	if awsutil.ErrorCodeIs(err, DuplicatePolicyAttachmentException) {
		err = nil
	}

	return err
}

func ListPolicies(
	ctx context.Context,
	cfg *awscfg.Config,
	policyType PolicyType,
) (summaries []PolicySummary, err error) {
	var nextToken *string
	for {
		out, err := cfg.Organizations().ListPolicies(ctx, &organizations.ListPoliciesInput{
			Filter:    policyType,
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, out.Policies...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

// OrgAssumeRolePolicy returns an assume-role policy that will allow any
// principal in this organization to assume a role, which is useful for
// roles that don't necessarily know the account numbers and/or role names
// that are authorized to assume them.
func OrgAssumeRolePolicy(ctx context.Context, cfg *awscfg.Config) (*policies.Document, error) {
	org, err := cfg.DescribeOrganization(ctx)
	if err != nil {
		return nil, err
	}
	return &policies.Document{
		Statement: []policies.Statement{{
			Principal: &policies.Principal{AWS: []string{"*"}},
			Action:    []string{"sts:AssumeRole"},
			Condition: policies.Condition{"StringEquals": {
				"aws:PrincipalOrgID": []string{aws.ToString(org.Id)},
			}},
		}},
	}, nil
}

func attachPolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	policyId, rootId string,
) error {
	_, err := cfg.Organizations().AttachPolicy(ctx, &organizations.AttachPolicyInput{
		PolicyId: aws.String(policyId),
		TargetId: aws.String(rootId),
	})
	return err
}

func createPolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	policyType PolicyType,
	doc *policies.Document,
) (*Policy, error) {
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	out, err := cfg.Organizations().CreatePolicy(ctx, &organizations.CreatePolicyInput{
		Content:     aws.String(docJSON),
		Description: aws.String(""),
		Name:        aws.String(name),
		Type:        policyType,
	})
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}

func enablePolicyType(
	ctx context.Context,
	cfg *awscfg.Config,
	rootId string,
	policyType PolicyType,
) error {
	_, err := cfg.Organizations().EnablePolicyType(ctx, &organizations.EnablePolicyTypeInput{
		PolicyType: policyType,
		RootId:     aws.String(rootId),
	})
	return err
}

func updatePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	policyId, name string,
	doc *policies.Document,
) (*Policy, error) {
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	out, err := cfg.Organizations().UpdatePolicy(ctx, &organizations.UpdatePolicyInput{
		Content:     aws.String(docJSON),
		Description: aws.String(""),
		Name:        aws.String(name),
		PolicyId:    aws.String(policyId),
	})
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}
