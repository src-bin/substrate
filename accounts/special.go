package accounts

import (
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/version"
)

const SpecialAccountsFilename = "substrate.special-accounts.json"

type SpecialAccountsDocument struct {
	Admonition jsonutil.Admonition `json:"#"`

	ManagementAccountId string

	AuditAccountId   string
	DeployAccountId  string
	NetworkAccountId string

	SubstrateVersion jsonutil.SubstrateVersion
}

func ReadSpecialAccountsDocument() (*SpecialAccountsDocument, error) {
	b, err := fileutil.ReadFile(SpecialAccountsFilename)
	if errors.Is(err, fs.ErrNotExist) {
		b = []byte("{}")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	d := &SpecialAccountsDocument{}
	if err := json.Unmarshal(b, d); err != nil {
		return nil, err
	}
	d.SubstrateVersion = jsonutil.SubstrateVersion(version.Version)
	return d, nil
}

func (d *SpecialAccountsDocument) Write() error {
	return jsonutil.Write(d, SpecialAccountsFilename)
}
