package awscfg

import (
	"fmt"
	"strings"

	"github.com/src-bin/substrate/accounts"
)

type AccountNotFound string

func NewAccountNotFound(tags ...string) error {
	if len(tags) == 3 && tags[0] == accounts.Admin && tags[1] == accounts.Admin {
		return AccountNotFound(fmt.Sprintf("%s/%s", accounts.Admin, tags[2]))
	}
	return AccountNotFound(strings.Join(tags, "/"))
}

func (err AccountNotFound) Error() string {
	return fmt.Sprintf("account not found: %s", string(err))
}

type OrganizationReaderError struct {
	error
	roleName string
}

func NewOrganizationReaderError(err error, roleName string) *OrganizationReaderError {
	return &OrganizationReaderError{err, roleName}
}

func (err *OrganizationReaderError) Err() error {
	return err.error
}

func (err *OrganizationReaderError) Error() string {
	target := "other roles"
	if err.roleName != "" {
		target = fmt.Sprintf("the %s role", err.roleName)
	}
	return fmt.Sprintf(
		"could not assume the OrganizationReader role in your organization's management account, which is a prerequisite for finding and assuming %s (actual error: %s)",
		target,
		err.Err(),
	)
}
