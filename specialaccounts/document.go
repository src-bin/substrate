package specialaccounts

import (
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/version"
)

const Filename = "substrate.special-accounts.json"

type Document struct {
	Admonition jsonutil.Admonition `json:"#"`

	ManagementAccountId string

	AuditAccountId   string
	DeployAccountId  string
	NetworkAccountId string

	SubstrateVersion jsonutil.SubstrateVersion

	CloudTrail struct {
		BucketName, TrailName string
		Manager               string // "Substrate" or "unknown"
	}
}

func ReadDocument() (*Document, error) {
	var b []byte
	pathname, err := fileutil.PathnameInParents(Filename)
	if err == nil {
		b, err = fileutil.ReadFile(pathname)
	}
	if errors.Is(err, fs.ErrNotExist) {
		b = []byte("{}")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	d := &Document{}
	if err := json.Unmarshal(b, d); err != nil {
		return nil, err
	}
	d.SubstrateVersion = jsonutil.SubstrateVersion(version.Version)
	return d, nil
}

func (d *Document) Write() error {
	return jsonutil.Write(d, Filename)
}
