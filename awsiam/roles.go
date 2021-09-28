package awsiam

import (
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

func AttachRolePolicy(
	svc *iam.IAM,
	roleName, policyArn string,
) error {
	in := &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	}
	_, err := svc.AttachRolePolicy(in)
	return err
}

func CreateRole(
	svc *iam.IAM,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
) (*Role, error) {
	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	in := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(docJSON),
		MaxSessionDuration:       aws.Int64(43200),
		// TODO permissionsBoundaryPolicyArn,
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	}
	out, err := svc.CreateRole(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return roleFromAPI(out.Role)
}

func CreateServiceLinkedRole(
	svc *iam.IAM,
	serviceName string,
) (*Role, error) {
	in := &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String(serviceName),
	}
	out, err := svc.CreateServiceLinkedRole(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return roleFromAPI(out.Role)
}

func DeleteRolePolicy(svc *iam.IAM, roleName string) error {
	in := &iam.DeleteRolePolicyInput{
		PolicyName: aws.String(SubstrateManaged),
		RoleName:   aws.String(roleName),
	}
	_, err := svc.DeleteRolePolicy(in)
	return err
}

func EnsureRole(
	svc *iam.IAM,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := CreateRole(
		svc,
		roleName,
		policies.AssumeRolePolicyDocument(&policies.Principal{Service: []string{"ec2.amazonaws.com"}}), // harmless solution to chicken and egg problem
		// TODO permissionsBoundaryPolicyArn,
	)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {

		// There was a time when Substrate created roles with the default
		// 1-hour maximum session duration. Lengthen that to 12 hours.
		if _, err := svc.UpdateRole(&iam.UpdateRoleInput{
			MaxSessionDuration: aws.Int64(43200),
			RoleName:           aws.String(roleName),
		}); err != nil {
			return nil, err
		}

		role, err = GetRole(svc, roleName)
	}
	if err != nil {
		return nil, err
	}

	if _, err := svc.TagRole(&iam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	}); err != nil {
		return nil, err
	}

	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := svc.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		RoleName:       aws.String(roleName),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func EnsureRoleWithPolicy(
	svc *iam.IAM,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
	doc *policies.Document,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := EnsureRole(svc, roleName, assumeRolePolicyDoc)
	if err != nil {
		return nil, err
	}

	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	in := &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(SubstrateManaged),
		RoleName:       aws.String(roleName),
	}
	if _, err := svc.PutRolePolicy(in); err != nil {
		return nil, err
	}

	return role, nil
}

func EnsureServiceLinkedRole(
	svc *iam.IAM,
	roleName, serviceName string, // not independent; must match AWS expectations
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := CreateServiceLinkedRole(svc, serviceName)
	if awsutil.ErrorCodeIs(err, InvalidInput) {
		role, err = GetRole(svc, roleName)
	}
	if err != nil {
		return nil, err
	}

	if _, err := svc.TagRole(&iam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func GetRole(svc *iam.IAM, roleName string) (*Role, error) {
	in := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}
	out, err := svc.GetRole(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return roleFromAPI(out.Role)
}

type Role struct {
	Arn              string
	AssumeRolePolicy *policies.Document
	Name             string
}

func roleFromAPI(role *iam.Role) (*Role, error) {
	s, err := url.PathUnescape(aws.StringValue(role.AssumeRolePolicyDocument))
	if err != nil {
		return nil, err
	}
	doc, err := policies.Unmarshal(s)
	return &Role{
		Arn:              aws.StringValue(role.Arn),
		AssumeRolePolicy: doc,
		Name:             aws.StringValue(role.RoleName),
	}, err
}

func (r *Role) AddPrincipal(svc *iam.IAM, principal *policies.Principal) error {
	if len(r.AssumeRolePolicy.Statement) == 0 {
		return fmt.Errorf("AssumeRolePolicy with zero Statements %+v", r.AssumeRolePolicy)
	}
	if len(r.AssumeRolePolicy.Statement) > 1 {
		return fmt.Errorf("AssumeRolePolicy with more than one Statement %+v", r.AssumeRolePolicy)
	}

	for _, s := range principal.AWS {
		r.AssumeRolePolicy.Statement[0].Principal.AWS.Add(s)
	}
	for _, s := range principal.Federated {
		r.AssumeRolePolicy.Statement[0].Principal.Federated.Add(s)
	}
	for _, s := range principal.Service {
		r.AssumeRolePolicy.Statement[0].Principal.Service.Add(s)
	}

	docJSON, err := r.AssumeRolePolicy.Marshal()
	if err != nil {
		return err
	}
	_, err = svc.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		RoleName:       aws.String(r.Name),
	})
	return err
}
