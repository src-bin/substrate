package intranetzip

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	base64sha256                             = new(bool)
	format, formatFlag, formatCompletionFunc = cmdutil.FormatFlag(
		cmdutil.FormatText,
		[]cmdutil.Format{cmdutil.FormatJSON, cmdutil.FormatText},
	)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "intranet-zip [--base64sha256] [--format <format>]",
		Hidden: true,
		Short:  "TODO intranetzip.Command().Short",
		Long:   `TODO intranetzip.Command().Long`,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"--base64sha256", "--format"}, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().BoolVar(base64sha256, "base64sha256", false, "print the base-64-encoded, SHA256 sum of the substrate-intranet binary instead of the binary itself (useful for rectifying lambda:UpdateFunctionCode API arguments and Terraform plans)")
	cmd.Flags().AddFlag(formatFlag)
	cmd.RegisterFlagCompletionFunc(formatFlag.Name, formatCompletionFunc)
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {

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
		switch *format {
		case cmdutil.FormatJSON:
			fmt.Println(jsonutil.MustString(terraformExternal{enc}))
		case cmdutil.FormatText:
			fmt.Println(enc)
		default:
			ui.Fatal(cmdutil.FormatFlagError(*format))
		}
		return
	}

	// Without --base64sha256, --format is nonsense since the default mode is
	// to write an ELF binary to standard output. Thus, any value but its
	// default should be rejected.
	if *format != cmdutil.FormatText {
		ui.Fatal("can't specify --format without --base64sha256")
	}

	// By default write the substrate-intranet binary to standard output.
	os.Stdout.Write(SubstrateIntranetZip)

}

//go:generate make -C ../../.. go-generate-intranet
//go:generate touch -t 202006100000.00 bootstrap
//go:generate zip -X substrate-intranet.zip bootstrap
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte

// terraformExternal is a type used to wrap the base-64-encoded SHA256 sum of
// the substrate-intranet binary in enough JSON structure that it can be
// queried from Terraform. This JSON output is used in
// modules/terraform/intranet/regional/main.tf.
type terraformExternal struct {
	Base64SHA256 string `json:"base64sha256"` // make it make sense in Terraform
}
