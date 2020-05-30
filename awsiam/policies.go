package awsiam

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

const FullAccess = "FullAccess"

// NOT DONE!
func EnsurePolicy(svc *iam.IAM, name, content string) error {

	policy, err := createPolicy(svc, name, content)
	/*
		if awsutil.ErrorCodeIs(err, DuplicatePolicyException) {
			err = nil

			for policySummary := range ListPolicies(svc, policyType) {
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
	*/
	log.Printf("%+v", policy)

	return err
}

func attachUserPolicy(svc *iam.IAM, username, policyArn string) error {
	in := &iam.AttachUserPolicyInput{
		PolicyArn: aws.String(policyArn),
		UserName:  aws.String(username),
	}
	_, err := svc.AttachUserPolicy(in)
	/*
		if awsutil.ErrorCodeIs(err, DuplicatePolicyAttachmentException) {
			err = nil
		}
	*/
	return err
}

func createPolicy(svc *iam.IAM, name, content string) (*iam.Policy, error) {
	in := &iam.CreatePolicyInput{
		PolicyDocument: aws.String(content),
		PolicyName:     aws.String(name),
	}
	out, err := svc.CreatePolicy(in)
	if err != nil {
		return nil, err
	}
	return out.Policy, nil
}
