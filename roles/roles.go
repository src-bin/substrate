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
	Intranet                      = "Intranet"
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

type ArnError string

func (err ArnError) Error() string {
	return fmt.Sprintf(
		"ArnError: %s isn't an anticipated format for an AWS IAM role ARN",
		string(err),
	)
}

func Name(roleArn string) (string, error) {
	parsed, err := arn.Parse(roleArn)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(parsed.Resource, "assumed-role/") {
		return strings.Split(parsed.Resource, "/")[1], nil
	}
	if strings.HasPrefix(parsed.Resource, "role/") {
		return strings.TrimPrefix(parsed.Resource, "role/"), nil
	}
	return "", ArnError(roleArn)
}
