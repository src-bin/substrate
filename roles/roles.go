package roles

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
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

func Name(roleArn string) (string, error) {
	parsed, err := arn.Parse(roleArn)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(parsed.Resource, "role/"), nil
}
