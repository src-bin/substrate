package roles

import "fmt"

const (
	NetworkAdministrator          = "NetworkAdministrator"
	OrganizationAccountAccessRole = "OrganizationAccountAccessRole"
	OrganizationAdministrator     = "OrganizationAdministrator"
	OrganizationReader            = "OrganizationReader"
)

func ARN(accountId, rolename string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, rolename)
}
