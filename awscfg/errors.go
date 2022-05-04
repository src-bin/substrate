package awscfg

import "fmt"

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
