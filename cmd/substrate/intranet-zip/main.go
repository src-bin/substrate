package intranetzip

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

//go:generate make -C ../../.. go-generate-intranet
//go:generate touch -t 202006100000.00 bootstrap
//go:generate zip -X substrate-intranet.zip bootstrap
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	base64sha256 := flag.Bool("base64sha256", false, "print the base-64-encoded, SHA256 sum of the substrate-intranet binary instead of the binary itself (useful for rectifying lambda:UpdateFunctionCode API arguments and Terraform plans)")
	format := cmdutil.SerializationFormatFlag(
		cmdutil.SerializationFormatText,
		`with -base64sha256, "text" for plaintext or "json" for JSON structured to work with Terraform's "external" data source`,
	)
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
		sum := sha256.Sum256(SubstrateIntranetZip)
		enc := base64.StdEncoding.EncodeToString(sum[:])
		switch format.String() {
		case cmdutil.SerializationFormatJSON:
			fmt.Println(jsonutil.MustString(terraformExternal{enc}))
		case cmdutil.SerializationFormatText:
			fmt.Println(enc)
		default:
			ui.Fatalf("-format %q not supported", format)
		}
		return
	}

	// Without -base64sha256, -format is nonsense since the default mode is
	// to write an ELF binary to standard output. Thus, any value but its
	// default should be rejected.
	if format.String() != cmdutil.SerializationFormatText {
		ui.Fatal(`can't specify -format "..." without -base64sha256`)
	}

	// By default write the substrate-intranet binary to standard output.
	os.Stdout.Write(SubstrateIntranetZip)

}

// terraformExternal is a type used to wrap the base-64-encoded SHA256 sum of
// the substrate-intranet binary in enough JSON structure that it can be
// queried from Terraform. This JSON output is used in
// modules/terraform/intranet/regional/main.tf.
type terraformExternal struct {
	Base64SHA256 string `json:"base64sha256"` // make it make sense in Terraform
}
