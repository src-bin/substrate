package naming

import (
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const (
	IntranetDNSDomainNameFilename = "substrate.intranet-dns-domain-name"
	IntranetDNSDomainNameVariable = "SUBSTRATE_INTRANET" // XXX or just "SUBSTRATE"?
)

func IntranetDNSDomainName() string {
	pathname, err := fileutil.PathnameInParents(IntranetDNSDomainNameFilename)
	if err != nil {
		ui.Fatalf("substrate.* not found in this or any parent directory; change to your Substrate repository or set SUBSTRATE_ROOT to its path in your environment (%v)", err)
	}
	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		ui.Fatal(err)
	}
	return string(fileutil.Tidy(b))
}
