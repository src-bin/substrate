package awsiam

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

func AttachRolePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName, policyARN string,
) error {
	_, err := cfg.IAM().AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		PolicyArn: aws.String(policyARN),
		RoleName:  aws.String(roleName),
	})
	return err
}

func CreateRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyARN,
) (*Role, error) {
	if os.Getenv("SUBSTRATE_DEBUG_AWS_IAM_ASSUME_ROLE_POLICIES") != "" {
		ui.Printf(
			"assume-role policy document for %s in account number %s: %s",
			roleName,
			cfg.MustAccountId(ctx),
			jsonutil.MustString(assumeRolePolicyDoc),
		)
	}

	docJSON, err := assumeRolePolicyDoc.Marshal()
	if err != nil {
		return nil, err
	}
	out, err := cfg.IAM().CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(docJSON),
		MaxSessionDuration:       aws.Int32(43200),
		// TODO permissionsBoundaryPolicyARN,
		RoleName: aws.String(roleName),
		Tags:     tagsFor(roleName),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return roleFromAPI(ctx, cfg, out.Role)
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
	return roleFromAPI(ctx, cfg, out.Role)
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

// DeleteRoleWithConfirmation is a higher-level way to delete a role that
// checks to see if the role even exists, confirms the deletion, and then
// deletes not only the role but also the instance profile and inline
// policies that must be detached and/or deleted first.
func DeleteRoleWithConfirmation(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	force bool,
) error {

	// If there's no role to delete, don't bother confirming and don't
	// bother printing any progress indication.
	role, err := GetRole(ctx, cfg, roleName)
	if awsutil.ErrorCodeIs(err, NoSuchEntity) {
		return nil
	} else if err != nil {
		return err
	}

	// This seems annoyingly superfluous but since we're confirming whether
	// we should delete a potentially critical role, we should really give
	// them the best possible information.
	mgmtCfg, err := cfg.OrganizationReader(ctx)
	if err != nil {
		return err
	}
	accountId, err := cfg.AccountId(ctx)
	if err != nil {
		return err
	}
	account, err := awsorgs.DescribeAccount(ctx, mgmtCfg, accountId)
	if err != nil {
		return err
	}

	// Only offer to delete Substrate-managed roles.
	if role.Tags[tagging.Manager] != tagging.Substrate {
		return nil
	}

	// There's a role to delete. Confirm before proceeding unless we've been
	// told to force it.
	if !force {
		ok, err := ui.Confirmf("delete role %s in %s? (yes/no)", roleName, account)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ui.Spinf("deleting role %s in %s", roleName, account)
	err = DeleteInstanceProfile(ctx, cfg, roleName)
	if err != nil && !awsutil.ErrorCodeIs(err, NoSuchEntity) {
		return err
	}
	err = DeleteRolePolicy(ctx, cfg, roleName)
	if err != nil && !awsutil.ErrorCodeIs(err, NoSuchEntity) {
		return err
	}
	err = DeleteRole(ctx, cfg, roleName)
	if err != nil && !awsutil.ErrorCodeIs(err, NoSuchEntity) {
		return err
	}
	ui.Stop("ok")

	return nil
}

func EnsureRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	assumeRolePolicyDoc *policies.Document,
	// TODO permissionsBoundaryPolicyARN,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded
	client := cfg.IAM()

	role, err := CreateRole(
		ctx,
		cfg,
		roleName,
		policies.AssumeRolePolicyDocument(&policies.Principal{Service: []string{"ec2.amazonaws.com"}}), // harmless solution to chicken and egg problem
		// TODO permissionsBoundaryPolicyARN,
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
	// TODO permissionsBoundaryPolicyARN,
	doc *policies.Document,
) (*Role, error) {
	defer time.Sleep(1e9) // avoid Throttling: Rate exceeded

	role, err := EnsureRole(ctx, cfg, roleName, assumeRolePolicyDoc)
	if err != nil {
		return nil, err
	}

	if err := PutRolePolicy(ctx, cfg, roleName, SubstrateManaged, doc); err != nil {
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
	return roleFromAPI(ctx, cfg, out.Role)
}

func ListAttachedRolePolicies(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
) ([]string, error) {
	var (
		arns   []string
		marker *string
	)
	for {
		out, err := cfg.IAM().ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			Marker:   marker,
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return nil, err
		}
		for _, attachment := range out.AttachedPolicies {
			arns = append(arns, aws.ToString(attachment.PolicyArn))
		}
		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}
	return arns, nil
}

func ListRoles(ctx context.Context, cfg *awscfg.Config) ([]*Role, error) {
	out, err := cfg.IAM().ListRoles(ctx, &iam.ListRolesInput{})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	roles := make([]*Role, len(out.Roles))
	for i := 0; i < len(out.Roles); i++ {
		roles[i], err = roleFromAPI(ctx, cfg, &out.Roles[i])
		if err != nil {
			return nil, err
		}
	}
	//log.Printf("%+v", roles)
	return roles, nil
}

func PutRolePolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName, policyName string,
	doc *policies.Document,
) error {
	docJSON, err := doc.Marshal()
	if err != nil {
		return err
	}
	_, err = cfg.IAM().PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(policyName),
		RoleName:       aws.String(roleName),
	})
	return err
}

type Role struct {
	ARN              string
	AssumeRolePolicy *policies.Document
	Name             string
	Tags             tagging.Map
}

func roleFromAPI(ctx context.Context, cfg *awscfg.Config, role *types.Role) (*Role, error) {
	name := aws.ToString(role.RoleName)

	s, err := url.PathUnescape(aws.ToString(role.AssumeRolePolicyDocument))
	if err != nil {
		return nil, err
	}
	doc, err := policies.UnmarshalString(s)
	if err != nil {
		return nil, err
	}

	tags, err := ListRoleTags(ctx, cfg, name)
	if err != nil {
		return nil, err
	}

	return &Role{
		ARN:              aws.ToString(role.Arn),
		AssumeRolePolicy: doc,
		Name:             name,
		Tags:             tags,
	}, err
}
