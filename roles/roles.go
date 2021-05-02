package roles

import (
	"fmt"
)

const (
	Administrator                 = "Administrator"
	Auditor                       = "Auditor"
	DeployAdministrator           = "DeployAdministrator"
	NetworkAdministrator          = "NetworkAdministrator"
	OrganizationAccountAccessRole = "OrganizationAccountAccessRole"
	OrganizationAdministrator     = "OrganizationAdministrator"
	OrganizationReader            = "OrganizationReader"
	SubstrateCredentialFactory    = "substrate-credential-factory"
	TerraformStateManager         = "TerraformStateManager"
)

func Arn(accountId, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, roleName)
}
