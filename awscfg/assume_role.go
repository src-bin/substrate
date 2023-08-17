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
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

// AssumeManagementRole assumes the given role in the organization's
// management account. It can only be called on a *Config with the
// Administrator role in an admin account, Substrate role or user in the
// Substrate account, or the OrganizationAdministrator or Substrate role or
// user already in the management account. The duration parameter is limited
// to an AWS-enforced maximum of 3600 when the *Config receiving the call has
// a role instead of a user, which is called role chaining. See
// <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
// for more information.
func (c *Config) AssumeManagementRole(
	ctx context.Context,
	roleName string,
	duration time.Duration,
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

	var mgmtAccountId string
	if org, err := c.DescribeOrganization(ctx); err == nil {
		//log.Print(jsonutil.MustString(org))
		mgmtAccountId = aws.ToString(org.MasterAccountId)
	} else if awsutil.ErrorCodeIs(err, AWSOrganizationsNotInUseException) {
		mgmtAccountId = aws.ToString(callerIdentity.Account)
	} else {
		return nil, err
	}
	if err := EnsureManagementAccountIdMatchesDisk(mgmtAccountId); err != nil {
		return nil, err
	}

	// Return early if we're already in the management account and already have
	// a sufficiently capable role, even if it's not exactly as requested. If
	// we're only seeking OrganizationReader, the more capable
	// OrganizationAdministrator and Substrate roles and users will suffice.
	// And if we're seeking the OrganizationAdministrator or Substrate roles,
	// either of those roles or the users by the same names will suffice.
	if aws.ToString(callerIdentity.Account) == mgmtAccountId {
		switch roleName {
		case roles.OrganizationReader:
			if a.Resource == path.Join("role", roles.OrganizationReader) {
				return c.Copy(), nil
			}
			fallthrough
		case roles.OrganizationAdministrator, roles.Substrate:
			switch a.Resource {
			case path.Join("role", users.OrganizationAdministrator),
				path.Join("role", users.Substrate),
				path.Join("user", users.OrganizationAdministrator),
				path.Join("user", users.Substrate):
				return c.Copy(), nil
			}
		}
	}

	// As a transitional step, if we're trying to assume the Substrate role
	// and there's an error, try again with OrganizationAdministrator.
	if roleName == roles.Substrate {
		cfg, err := c.AssumeRole(ctx, mgmtAccountId, roleName, duration)
		if err == nil {
			return cfg, nil
		}
		ui.Print("falling back from the Substrate role to the OrganizationAdministrator role")
		roleName = roles.OrganizationAdministrator // TODO restrict to more specific error cases
	}

	return c.AssumeRole(ctx, mgmtAccountId, roleName, duration)
}

// AssumeRole assumes the given role in the given account and returns a new
// *Config there. It can be called on any *Config but is most often (and most
// effectively) called on one with the Administrator role in an admin account,
// the Substrate role or user in the Substrate account, or the
// OrganizationAdministrator or Substrate role or user in the management
// account.
func (c *Config) AssumeRole(
	ctx context.Context,
	accountId string,
	roleName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/> is 1 hour
) (*Config, error) {
	//ui.Printf("assuming %s in %s", roleName, accountId)
	if roleName != roles.OrganizationReader {
		c.event.SetFinalAccountId(accountId)
		c.event.SetFinalRoleName(roleName)
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

func (c *Config) AssumeRoleARN(ctx context.Context, roleARN string, duration time.Duration) (*Config, error) {
	parsed, err := arn.Parse(roleARN)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(parsed.Resource, "role/") {
		return nil, roles.ARNError(roleARN)
	}
	return c.AssumeRole(ctx, parsed.AccountID, strings.TrimPrefix(parsed.Resource, "role/"), duration)
}

// AssumeServiceRole assumes the given role in the service account identified
// by the given domain, environment, and quality. It can be called on any
// *Config but is most often (and most effectively) called on one with the
// Administrator role in an admin account, the Substrate role or user in the
// Substrate account, or the OrganizationAdministrator or Substrate role or
// user in the management account.
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
// called on one with the Administrator role in an admin account, the Substrate
// role or user in the Substrate account, or the OrganizationAdministrator  or
// Substrate role or user in the management account.
func (c *Config) AssumeSpecialRole(
	ctx context.Context,
	name string,
	roleName string,
	duration time.Duration,
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

// AssumeSubstrateRole assumes the given role in the Substrate account. It
// can be called on any *Config but is most often (and most effectively)
// called from Administrator or Substrate in the Substrate account or
// OrganizationAdministrator or Substrate in the management account. It's
// for times when you want to be sure you're in the Substrate account.
func (c *Config) AssumeSubstrateRole(
	ctx context.Context,
	roleName string,
	duration time.Duration,
) (*Config, error) {
	account, err := c.FindSubstrateAccount(ctx)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, NewAccountNotFound(naming.Substrate)
	}
	//log.Print(jsonutil.MustString(account))
	return c.AssumeRole(ctx, aws.ToString(account.Id), roleName, duration)
}
