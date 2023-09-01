package naming

import (
	"fmt"
	"os"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const (
	IntranetDNSDomainNameFilename = "substrate.intranet-dns-domain-name"
	IntranetDNSDomainNameVariable = "SUBSTRATE_INTRANET" // XXX or just "SUBSTRATE"?
)

func IntranetDNSDomainName() (string, error) {
	pathname, err := fileutil.PathnameInParents(IntranetDNSDomainNameFilename)
	if err != nil {
		return "", fmt.Errorf("substrate.* not found in this or any parent directory; change to your Substrate repository or set SUBSTRATE_ROOT to its path in your environment (%v)", err)
	}
	b, err := os.ReadFile(pathname)
	if err != nil {
		return "", err
	}
	return string(fileutil.Tidy(b)), nil
}

func MustIntranetDNSDomainName() string {
	s, err := IntranetDNSDomainName()
	ui.Must(err)
	return s
}
