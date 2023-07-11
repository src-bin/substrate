package users

import "fmt"

const (
	CredentialFactory         = "CredentialFactory"         // legacy
	OrganizationAdministrator = "OrganizationAdministrator" // legacy
	Substrate                 = "Substrate"
)

func ARN(accountId, username string) string {
	return fmt.Sprintf("arn:aws:iam::%s:user/%s", accountId, username)
}
