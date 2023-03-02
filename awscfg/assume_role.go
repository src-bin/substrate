package awscfg

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/users"
)

// AssumeAdminRole assumes the given role in the admin account with the given
// quality. It should only be called on a *Config with the
// OrganizationAdministrator role or user in the management account.
func (c *Config) AssumeAdminRole(
	ctx context.Context,
	quality string,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	return c.AssumeServiceRole(ctx, naming.Admin, naming.Admin, quality, roleName, duration)
}

// AssumeManagementRole assumes the given role in the organization's
// management account. It can only be called on a *Config with the
// Administrator role in an admin account or one already in the
// management account.
func (c *Config) AssumeManagementRole(
	ctx context.Context,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {

	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	//log.Print(jsonutil.MustString(callerIdentity))
	a, err := arn.Parse(aws.ToString(callerIdentity.Arn))
	if err != nil {
		return nil, err
	}

	// Return early if we're the OrganizationAdministrator user, which is
	// equivalent to the OrganizationAdministrator role in every meaningful
	// way and is guaranteed to exist early enough to be used even during the
	// very first run of `substrate bootstrap-management-account`.
	if roleName == roles.OrganizationAdministrator && a.Resource == path.Join("user", users.OrganizationAdministrator) {
		return c.Copy(), nil
	}

	// Similarly return early if we're the OrganizationAdministrator user
	// or role or indeed the OrganizationReader role and all we're trying
	// to get to is the OrganzationReader role.
	if roleName == roles.OrganizationReader && (a.Resource == path.Join("user", users.OrganizationAdministrator) || a.Resource == path.Join("role", users.OrganizationAdministrator)) {
		return c.Copy(), nil
	}

	org, err := c.DescribeOrganization(ctx)
	if err != nil {
		return nil, err
	}
	mgmtAccountId := aws.ToString(org.MasterAccountId)
	//log.Print(jsonutil.MustString(org))
	if err := EnsureManagementAccountIdMatchesDisk(mgmtAccountId); err != nil {
		return nil, err
	}

	return c.AssumeRole(ctx, mgmtAccountId, roleName, duration)
}

// AssumeRole assumes the given role in the given account and returns a new
// *Config there. It can be called on any *Config but is most often (and most
// effectively) called on one with the Administrator role in an admin account
// or the OrganizationAdministrator role or user in the management account.
func (c *Config) AssumeRole(
	ctx context.Context,
	accountId string,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/> is 1 hour
) (*Config, error) {
	//ui.Printf("assuming %s in %s", roleName, accountId)
	if roleName != roles.OrganizationReader {
		c.event.FinalAccountId = accountId
		c.event.FinalRoleName = roleName
	}

	// Return early if we're already roleName in accountId. This is critical
	// in the new world (beginning in 2022.10) in which it's no longer implied
	// that a role may assume itself.
	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	callerRoleName, err := roles.Name(aws.ToString(callerIdentity.Arn))
	if err == nil {
		if aws.ToString(callerIdentity.Account) == accountId && callerRoleName == roleName {
			//ui.Printf("short-circuiting while assuming %s in %s", roleName, accountId)
			return c, nil
		}
	} else if err != nil {
		if _, ok := err.(roles.ARNError); !ok {
			return nil, err
		}
	}

	roleSessionName := contextutil.ValueString(ctx, contextutil.Username)
	if roleSessionName == "" {
		safeSubcommand, _, _ := strings.Cut(
			strings.TrimPrefix(
				contextutil.ValueString(ctx, contextutil.Subcommand),
				"/",
			),
			"/",
		)
		roleSessionName = fmt.Sprintf(
			"%s-%s",
			contextutil.ValueString(ctx, contextutil.Command),
			safeSubcommand,
		)
		if roleSessionName == "-" {
			roleSessionName = filepath.Base(os.Args[0])
		}
	}
	if len(roleSessionName) > 64 {
		roleSessionName = roleSessionName[:64]
	}

	cfg := &Config{
		cfg:               c.cfg.Copy(),
		deferredTelemetry: c.deferredTelemetry, // better twice than not at all
		event:             c.event,
	}

	cfg.cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(
		c.STS(),
		roles.ARN(accountId, roleName),
		func(options *stscreds.AssumeRoleOptions) {
			options.Duration = duration
			options.RoleSessionName = roleSessionName
		},
	))

	callerIdentity, err = cfg.WaitUntilCredentialsWork(ctx)
	//log.Print(jsonutil.MustString(callerIdentity))

	// TODO detect when we're Administrator (or whatever) in a service account
	// and trying to assume OrganizationAdministrator in the management
	// account, which will have just failed repeatedly, to provide an actually
	// helpful error message. It may be possible to push such an error further
	// up the stack so that we get it faster, too, but that seems dangerously
	// close to reimplementing part of IAM which is a bad idea.
	//
	// This will specifically improve the experience I had on a demo call
	// during which I forgot I'd assumed a role and needed to unassume-role
	// before carrying on.

	return cfg, err
}

// AssumeServiceRole assumes the given role in the service account identified
// by the given domain, environment, and quality. It can be called on any
// *Config but is most often (and most effectively) called on one with the
// Administrator role in an admin account or the OrganizationAdministrator
// role or user in the management account.
func (c *Config) AssumeServiceRole(
	ctx context.Context,
	domain, environment, quality string,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	account, err := c.FindServiceAccount(ctx, domain, environment, quality)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, NewAccountNotFound(domain, environment, quality)
	}
	//log.Print(jsonutil.MustString(account))
	return c.AssumeRole(ctx, aws.ToString(account.Id), roleName, duration)
}

// AssumeSpecialRole assumes the given role in the named special account. It
// can be called on any *Config but is most often (and most effectively)
// called on one with the Administrator role in an admin account or the
// OrganizationAdministrator role or user in the management account.
func (c *Config) AssumeSpecialRole(
	ctx context.Context,
	name string,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	account, err := c.FindAccount(ctx, func(a *Account) bool {
		//log.Print(jsonutil.MustString(a))
		return aws.ToString(a.Name) == name
	})
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, NewAccountNotFound(name)
	}
	//log.Print(jsonutil.MustString(account))
	return c.AssumeRole(ctx, aws.ToString(account.Id), roleName, duration)
}
