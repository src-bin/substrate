package awscfg

import (
	"fmt"
	"io/ioutil"

	"github.com/src-bin/substrate/fileutil"
)

const (
	ManagementAccountIdFilename    = "substrate.management-account-id"
	OldManagementAccountIdFilename = "substrate.master-account-id"
)

func EnsureManagementAccountIdMatchesDisk(managementAccountId string) error {

	// We'll never have this file when we're e.g. in Lambda.
	pathname, err := fileutil.PathnameInParents(ManagementAccountIdFilename)
	if err != nil {
		return nil
	}

	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		return err
	}
	if diskManagementAccountId := fileutil.Tidy(b); managementAccountId != diskManagementAccountId {
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

func WriteManagementAccountIdToDisk(managementAccountId string) error {
	if !fileutil.Exists(ManagementAccountIdFilename) {
		if err := ioutil.WriteFile(ManagementAccountIdFilename, []byte(fmt.Sprintln(managementAccountId)), 0666); err != nil {
			return err
		}
	}
	return nil
}
