package awsiam

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

func AttachRolePolicy(
	svc *iam.IAM,
	rolename, policyArn string,
) error {
	in := &iam.AttachRolePolicyInput{
		RoleName:  aws.String(rolename),
		PolicyArn: aws.String(policyArn),
	}
	_, err := svc.AttachRolePolicy(in)
	return err
}

func CreateRole(
	svc *iam.IAM,
	rolename string,
	assumeRolePolicyDoc *policies.Document,
) (*Role, error) {
	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	in := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(docJSON),
		MaxSessionDuration:       aws.Int64(12 * 60 * 60),
		RoleName:                 aws.String(rolename),
		Tags:                     tagsFor(rolename),
	}
	out, err := svc.CreateRole(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return roleFromAPI(out.Role)
}

func EnsureRole(
	svc *iam.IAM,
	rolename string,
	assumeRolePolicyDoc *policies.Document,
) (*Role, error) {

	role, err := CreateRole(svc, rolename, assumeRolePolicyDoc)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		role, err = GetRole(svc, rolename)
	}
	if err != nil {
		return nil, err
	}

	if _, err := svc.TagRole(&iam.TagRoleInput{
		RoleName: aws.String(rolename),
		Tags:     tagsFor(rolename),
	}); err != nil {
		return nil, err
	}

	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := svc.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		RoleName:       aws.String(rolename),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func EnsureRoleWithPolicy(
	svc *iam.IAM,
	rolename string,
	assumeRolePolicyDoc *policies.Document,
	doc *policies.Document,
) (*Role, error) {

	role, err := EnsureRole(svc, rolename, assumeRolePolicyDoc)
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
		RoleName:       aws.String(rolename),
	}
	if _, err := svc.PutRolePolicy(in); err != nil {
		return nil, err
	}

	return role, nil
}

func GetRole(svc *iam.IAM, rolename string) (*Role, error) {
	in := &iam.GetRoleInput{
		RoleName: aws.String(rolename),
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
