package awsorgs

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsutil"
)

type PolicyType string

const (
	DuplicatePolicyAttachmentException            = "DuplicatePolicyAttachmentException"
	DuplicatePolicyException                      = "DuplicatePolicyException"
	PolicyTypeAlreadyEnabledException             = "PolicyTypeAlreadyEnabledException"
	SERVICE_CONTROL_POLICY             PolicyType = "SERVICE_CONTROL_POLICY"
	TAG_POLICY                         PolicyType = "TAG_POLICY"
)

// EnsurePolicy makes potentially several AWS API requests to ensure,
// regardless of initial state, that a policy by a given name and type exists,
// has the desired content, and is attached to the specified root.
//
// A curiosity:  Though DescribeOrganization alludes to the ability to attach
// service control policies to the master account, that does not appear to be
// possible (without first enabling service control policies).  It's unclear
// if such policies would apply only to that account if attached there instead
// of the root.  In other words, the distinction between a master account and
// the root is murkier than I thought yesterday.
func EnsurePolicy(
	svc *organizations.Organizations,
	root *organizations.Root,
	name string,
	policyType PolicyType,
	content string,
) error {
	err := enablePolicyType(svc, aws.StringValue(root.Id), policyType)
	if awsutil.ErrorCodeIs(err, PolicyTypeAlreadyEnabledException) {
		err = nil
	}
	if err != nil {
		return err
	}

	policy, err := createPolicy(svc, name, policyType, content)
	if awsutil.ErrorCodeIs(err, DuplicatePolicyException) {
		err = nil

		for _, policySummary := range ListPolicies(svc, policyType) {
			if aws.StringValue(policySummary.Name) != name {
				continue
			}

			policy, err = updatePolicy(svc, aws.StringValue(policySummary.Id), name, content)
			if err != nil {
				return err
			}
		}

	}
	if err != nil {
		return err
	}
	//log.Printf("%+v", policy)

	// TODO tag the policy

	err = attachPolicy(svc, aws.StringValue(policy.PolicySummary.Id), aws.StringValue(root.Id))
	if awsutil.ErrorCodeIs(err, DuplicatePolicyAttachmentException) {
		err = nil
	}

	return err
}

func ListPolicies(
	svc *organizations.Organizations,
	policyType PolicyType,
) (policySummaries []*organizations.PolicySummary) {
	var nextToken *string
	for {
		in := &organizations.ListPoliciesInput{
			Filter:    aws.String(string(policyType)),
			NextToken: nextToken,
		}
		out, err := svc.ListPolicies(in)
		if err != nil {
			log.Fatal(err)
		}
		policySummaries = append(policySummaries, out.Policies...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func attachPolicy(svc *organizations.Organizations, policyId, rootId string) error {
	in := &organizations.AttachPolicyInput{
		PolicyId: aws.String(policyId),
		TargetId: aws.String(rootId),
	}
	_, err := svc.AttachPolicy(in)
	return err
}

func createPolicy(svc *organizations.Organizations, name string, policyType PolicyType, content string) (*organizations.Policy, error) {
	in := &organizations.CreatePolicyInput{
		Content:     aws.String(content),
		Description: aws.String(""),
		Name:        aws.String(name),
		Type:        aws.String(string(policyType)),
	}
	out, err := svc.CreatePolicy(in)
	return out.Policy, err
}

func enablePolicyType(svc *organizations.Organizations, rootId string, policyType PolicyType) error {
	in := &organizations.EnablePolicyTypeInput{
		PolicyType: aws.String(string(policyType)),
		RootId:     aws.String(rootId),
	}
	_, err := svc.EnablePolicyType(in)
	return err
}

func updatePolicy(svc *organizations.Organizations, policyId, name, content string) (*organizations.Policy, error) {
	in := &organizations.UpdatePolicyInput{
		Content:     aws.String(content),
		Description: aws.String(""),
		Name:        aws.String(name),
		PolicyId:    aws.String(policyId),
	}
	out, err := svc.UpdatePolicy(in)
	return out.Policy, err
}
