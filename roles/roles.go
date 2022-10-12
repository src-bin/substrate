package roles

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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
	TerraformStateManager         = "TerraformStateManager"
)

func ARN(accountId, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, roleName)
}

type ARNError string

func (err ARNError) Error() string {
	return fmt.Sprintf(
		"ARNError: %s isn't an anticipated format for an AWS IAM role ARN",
		string(err),
	)
}

func Name(roleARN string) (string, error) {
	parsed, err := arn.Parse(roleARN)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(parsed.Resource, "assumed-role/") {
		return strings.Split(parsed.Resource, "/")[1], nil
	}
	if strings.HasPrefix(parsed.Resource, "role/") {
		return strings.TrimPrefix(parsed.Resource, "role/"), nil
	}

	// There's an OrganizationAdministrator IAM user to go along with the IAM
	// role to facilitate bootstrapping.
	if strings.HasPrefix(parsed.Resource, "user/") {
		name := strings.TrimPrefix(parsed.Resource, "user/")
		if name == OrganizationAdministrator {
			return name, nil
		}
	}

	return "", ARNError(roleARN)
}
