package intranetzip

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/src-bin/substrate/awscfg"
	createadminaccount "github.com/src-bin/substrate/cmd/substrate/create-admin-account"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	base64sha256 := flag.Bool("base64sha256", false, "print the base-64-encoded, SHA256 sum of the substrate-intranet binary instead of the binary itself (useful for rectifying lambda:UpdateFunctionCode API arguments and Terraform plans)")
	flag.Usage = func() {
		ui.Print("Usage: substrate intranet-zip")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Though this command doesn't create any AWS accounts or apply any
	// Terraform modules, which are the usual things we're eager to prevent
	// happening based on old code, it would be bad to downgrade a customer's
	// Intranet and this is an effective way to prevent that.
	versionutil.PreventDowngrade(ctx, cfg)

	// With -base64sha256, print the checksum of the binary we would have
	// printed without that option.
	if *base64sha256 {
		sum := sha256.Sum256(createadminaccount.SubstrateIntranetZip)
		fmt.Println(base64.RawStdEncoding.EncodeToString(sum[:]))
		return
	}

	// By default write the substrate-intranet binary to standard output.
	os.Stdout.Write(createadminaccount.SubstrateIntranetZip)

}
