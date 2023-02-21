package awsiam

import (
	"context"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

func AttachRolePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName, policyArn string,
) error {
	_, err := cfg.IAM().AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	return err
}

func CreateRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
) (*Role, error) {
	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	out, err := cfg.IAM().CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(docJSON),
		MaxSessionDuration:       aws.Int32(43200),
		// TODO permissionsBoundaryPolicyArn,
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return roleFromAPI(out.Role)
}

func CreateServiceLinkedRole(
	ctx context.Context,
	cfg *awscfg.Config,
	serviceName string,
) (*Role, error) {
	out, err := cfg.IAM().CreateServiceLinkedRole(ctx, &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String(serviceName),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return roleFromAPI(out.Role)
}

func DeleteRole(ctx context.Context, cfg *awscfg.Config, roleName string) error {
	_, err := cfg.IAM().DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	return err
}

func DeleteRolePolicy(ctx context.Context, cfg *awscfg.Config, roleName string) error {
	_, err := cfg.IAM().DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
		PolicyName: aws.String(SubstrateManaged),
		RoleName:   aws.String(roleName),
	})
	return err
}

func EnsureRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded
	client := cfg.IAM()

	role, err := CreateRole(
		ctx,
		cfg,
		roleName,
		policies.AssumeRolePolicyDocument(&policies.Principal{Service: []string{"ec2.amazonaws.com"}}), // harmless solution to chicken and egg problem
		// TODO permissionsBoundaryPolicyArn,
	)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {

		// There was a time when Substrate created roles with the default
		// 1-hour maximum session duration. Lengthen that to 12 hours.
		if _, err := client.UpdateRole(ctx, &iam.UpdateRoleInput{
			MaxSessionDuration: aws.Int32(43200),
			RoleName:           aws.String(roleName),
		}); err != nil {
			return nil, err
		}

		role, err = GetRole(ctx, cfg, roleName)
	}
	if err != nil {
		return nil, err
	}

	if _, err := client.TagRole(ctx, &iam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	}); err != nil {
		return nil, err
	}

	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := cfg.IAM().UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		RoleName:       aws.String(roleName),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func EnsureRoleWithPolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyArn,
	doc *policies.Document,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := EnsureRole(ctx, cfg, roleName, assumeRolePolicyDoc)
	if err != nil {
		return nil, err
	}

	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := cfg.IAM().PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(SubstrateManaged),
		RoleName:       aws.String(roleName),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func EnsureServiceLinkedRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName, serviceName string, // not independent; must match AWS expectations
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := CreateServiceLinkedRole(ctx, cfg, serviceName)
	if awsutil.ErrorCodeIs(err, InvalidInput) {
		role, err = GetRole(ctx, cfg, roleName)
	}
	if err != nil {
		return nil, err
	}

	if _, err := cfg.IAM().TagRole(ctx, &iam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	}); err != nil {
		return nil, err
	}

	return role, nil
}

func GetRole(ctx context.Context, cfg *awscfg.Config, roleName string) (*Role, error) {
	out, err := cfg.IAM().GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
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

func roleFromAPI(role *types.Role) (*Role, error) {
	s, err := url.PathUnescape(aws.ToString(role.AssumeRolePolicyDocument))
	if err != nil {
		return nil, err
	}
	doc, err := policies.UnmarshalString(s)
	return &Role{
		Arn:              aws.ToString(role.Arn),
		AssumeRolePolicy: doc,
		Name:             aws.ToString(role.RoleName),
	}, err
}
