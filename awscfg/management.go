package awscfg

import (
	"fmt"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/specialaccounts"
)

const (
	ManagementAccountIdFilename = "substrate.management-account-id"
)

func EnsureManagementAccountIdMatchesDisk(managementAccountId string) error {

	// We'll never have either of the files we're about to open when we're in
	// Lambda and that's OK because in Lambda there's also no possibility of
	// being in one organization's context but meaning to be in another.

	// First, look for the new file that tracks all the special accounts.
	// This will succeed even if the file doesn't exist and the document will
	// have a zero value, ready to be filled in (which we will not do here).
	doc, err := specialaccounts.ReadDocument()
	if err != nil {
		return err
	}
	diskManagementAccountId := doc.ManagementAccountId

	// Secondly, but only if the management account number in the document
	// is empty, look for the old file that contains only the management
	// account number.
	if diskManagementAccountId == "" {
		pathname, err := fileutil.PathnameInParents(ManagementAccountIdFilename)
		if err != nil {
			return nil
		}
		b, err := fileutil.ReadFile(pathname)
		if err != nil {
			return err
		}
		diskManagementAccountId = fileutil.Tidy(b)
	}

	if managementAccountId != diskManagementAccountId {
		return ManagementAccountMismatchError(fmt.Sprintf(
			"the calling account's management account is %s but this directory's management account is %s",
			managementAccountId,
			diskManagementAccountId,
		))
	}
	return nil
}

type ManagementAccountMismatchError string

func (err ManagementAccountMismatchError) Error() string {
	return string(err)
}
