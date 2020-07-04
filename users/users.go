package users

import "fmt"

const (
	CredentialFactory         = "CredentialFactory"
	OrganizationAdministrator = "OrganizationAdministrator"
)

func Arn(accountId, username string) string {
	return fmt.Sprintf("arn:aws:iam::%s:user/%s", accountId, username)
}
