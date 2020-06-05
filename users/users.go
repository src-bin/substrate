package users

import "fmt"

const OrganizationAdministrator = "OrganizationAdministrator"

func Arn(accountId, username string) string {
	return fmt.Sprintf("arn:aws:iam::%s:user/%s", accountId, username)
}
