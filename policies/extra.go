package policies

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/src-bin/substrate/jsonutil"
)

func ExtraAdministratorAssumeRolePolicy() (*Document, error) {
	var extra Document
	if err := jsonutil.Read(
		ExtraAdministratorAssumeRolePolicyFilename,
		&extra,
	); errors.Is(err, fs.ErrNotExist) {
		return &extra, nil
	} else if err != nil {
		return nil, fmt.Errorf("error processing %s: %v", ExtraAdministratorAssumeRolePolicyFilename, err) // TODO wrap or type
	}
	//log.Printf("%+v", extra)
	return &extra, nil
}

func ExtraAuditorAssumeRolePolicy() (*Document, error) {
	var extra Document
	if err := jsonutil.Read(
		ExtraAuditorAssumeRolePolicyFilename,
		&extra,
	); errors.Is(err, fs.ErrNotExist) {
		return &extra, nil
	} else if err != nil {
		return nil, fmt.Errorf("error processing %s: %v", ExtraAuditorAssumeRolePolicyFilename, err) // TODO wrap or type
	}
	//log.Printf("%+v", extra)
	return &extra, nil
}
