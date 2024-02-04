package awscfg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

type Account struct {
	types.Account
	Tags tagging.Map
}

func (a *Account) AdministratorRoleName() string {
	switch a.Tags[tagging.SubstrateType] {
	case naming.Audit:
		return roles.AuditAdministrator
	case naming.Deploy:
		return roles.DeployAdministrator
	case naming.Management:
		return roles.OrganizationAdministrator
	case naming.Network:
		return roles.NetworkAdministrator
	case naming.Substrate:
		return roles.Administrator
	}

	switch a.Tags[tagging.SubstrateSpecialAccount] {
	case naming.Audit:
		return roles.AuditAdministrator
	case naming.Deploy:
		return roles.DeployAdministrator
	case naming.Management:
		return roles.OrganizationAdministrator
	case naming.Network:
		return roles.NetworkAdministrator
	}

	return roles.Administrator
}

func (a *Account) Config(
	ctx context.Context,
	cfg *Config,
	roleName string,
	duration time.Duration,
) (*Config, error) {
	return cfg.AssumeRole(ctx, aws.ToString(a.Id), roleName, duration)
}

func (a *Account) Quality() (quality string, err error) {
	quality = a.Tags[tagging.Quality]
	if quality != "" {
		return
	}
	var qualities []string
	qualities, err = naming.Qualities()
	if err != nil {
		return
	}
	quality = qualities[0]
	if len(qualities) > 1 {
		ui.Printf(
			"found multiple qualities %s; choosing %s for your Substrate account (this is temporary and inconsequential)",
			strings.Join(qualities, ", "),
			quality,
		)
	}
	return
}

func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Account
		AdministratorRoleARN, AuditorRoleARN string
	}{
		Account:              *a,
		AdministratorRoleARN: roles.ARN(aws.ToString(a.Id), a.AdministratorRoleName()),
		AuditorRoleARN:       roles.ARN(aws.ToString(a.Id), roles.Auditor),
	})
}

func (a *Account) String() string {
	switch a.Tags[tagging.SubstrateType] {
	case naming.Management:
		return fmt.Sprintf("management account number %s", aws.ToString(a.Id))
	case naming.Substrate:
		return fmt.Sprintf("Substrate account number %s", aws.ToString(a.Id))
	}

	if special := a.Tags[tagging.SubstrateSpecialAccount]; special != "" {
		return fmt.Sprintf("%s account number %s", special, aws.ToString(a.Id))
	}

	domain := a.Tags[tagging.Domain]
	environment := a.Tags[tagging.Environment]
	quality := a.Tags[tagging.Quality]
	if domain == naming.Admin && quality != "" {
		return fmt.Sprintf("admin account number %s (Quality: %s)", aws.ToString(a.Id), quality)
	}
	if domain != "" && environment != "" && quality != "" {
		return fmt.Sprintf("service account number %s (Domain: %s, Environment: %s, Quality: %s)", aws.ToString(a.Id), domain, environment, quality)
	}

	return fmt.Sprintf("account number %s", aws.ToString(a.Id))
}
