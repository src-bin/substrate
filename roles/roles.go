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
	TerraformStateManager         = "TerraformStateManager"
)

func Arn(accountId, rolename string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, rolename)
}
