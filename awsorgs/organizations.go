package awsorgs

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const (
	AccountAlreadyRegisteredException = "AccountAlreadyRegisteredException"
	AlreadyInOrganizationException    = "AlreadyInOrganizationException"
	TooManyRequestsException          = "TooManyRequestsException"
)

type (
	Organization = types.Organization
	Root         = types.Root
)

func CreateOrganization(ctx context.Context, cfg *awscfg.Config) (*Organization, error) {
	out, err := cfg.Organizations().CreateOrganization(ctx, &organizations.CreateOrganizationInput{
		FeatureSet: types.OrganizationFeatureSetAll, // we want service control policies
	})
	if err != nil {
		return nil, err
	}
	time.Sleep(10e9) // give Organizations time to finish so that subsequent CreateAccount, etc. will work (TODO do it gracefully)
	return out.Organization, nil
}

func DescribeOrganization(ctx context.Context, cfg *awscfg.Config) (*Organization, error) {
	return cfg.DescribeOrganization(ctx)
}

func DescribeRoot(ctx context.Context, cfg *awscfg.Config) (*Root, error) { // DescribeRoot is a made-up name
	roots, err := listRoots(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if len(roots) != 1 {
		return nil, errors.New("ListRoots responded with more than one Root which AWS says is impossible")
	}
	root := roots[0] // don't leak the slice
	return &root, nil
}

func EnableAWSServiceAccess(ctx context.Context, cfg *awscfg.Config, servicePrincipal string) error {
	_, err := cfg.Organizations().EnableAWSServiceAccess(ctx, &organizations.EnableAWSServiceAccessInput{
		ServicePrincipal: aws.String(servicePrincipal),
	})
	return err
}

func RegisterDelegatedAdministrator(ctx context.Context, cfg *awscfg.Config, accountId, servicePrincipal string) error {
	if err := EnableAWSServiceAccess(ctx, cfg, servicePrincipal); err != nil {
		return err
	}
	_, err := cfg.Organizations().RegisterDelegatedAdministrator(ctx, &organizations.RegisterDelegatedAdministratorInput{
		AccountId:        aws.String(accountId),
		ServicePrincipal: aws.String(servicePrincipal),
	})
	if awsutil.ErrorCodeIs(err, AccountAlreadyRegisteredException) {
		err = nil
	}
	return err
}

func listRoots(ctx context.Context, cfg *awscfg.Config) (roots []Root, err error) {
	var nextToken *string
	for {
		out, err := cfg.Organizations().ListRoots(ctx, &organizations.ListRootsInput{NextToken: nextToken})
		if err != nil {
			return nil, err
		}
		roots = append(roots, out.Roots...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
