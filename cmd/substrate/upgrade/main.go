package upgrade

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

// A handy and hacky way to test upgrades for trial versions now that we
// support those.
//func init() { version.Commit = "trial" }

var yes, no = new(bool), new(bool)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [--yes|--no]",
		Short: "TODO upgrade.Command().Short",
		Long:  `TODO upgrade.Command().Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"--yes", "--no"}, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().BoolVar(yes, "yes", false, `answer "yes" when offered an upgrade to accept it without confirmation`)
	cmd.Flags().BoolVar(no, "no", false, `answer "no" when offered an upgrade; exits 1 if there was an upgrade available`)
	return cmd
}

func Main(context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {

	// Print the current version for context and to make the fact that we're
	// about to print the prefix make more sense. Then print the URL we'll
	// fetch to see if there's an upgrade available.
	version.Print()
	ui.Printf(
		"checking <%s> to see if there's a Substrate upgrade available",
		versionutil.UpgradeURL(),
	)

	// Fetch the upgrade URL to see if there's an upgrade available from this
	// version for this customer. Exit 0 if there's not.
	v, ok, err := versionutil.CheckForUpgrade()
	ui.Must(err)
	if !ok {
		ui.Print("there's no Substrate upgrade available yet")
		return
	}

	// Construct the download URL.
	u := versionutil.DownloadURL(v, runtime.GOOS, runtime.GOARCH)
	ui.Printf("there's a Substrate upgrade available at <%s>", u)

	// Exit 1 when there's an upgrade available and either the -no option was
	// given or the user answers "no" to the prompt.
	if *no {
		ui.Print("re-run this command without the -no option to install it")
		os.Exit(1)
	}
	if !*yes {
		if ok, err := ui.Confirmf("upgrade to Substrate %s? (yes/no)", v); err != nil {
			ui.Fatal(err)
		} else if !ok {
			ui.Print("not upgrading Substrate") // not ui.Fatal to suppress the stack trace
			os.Exit(1)
		}
	}

	// Be pretty sure the directory where substrate is stored is writable.
	dirname, err := cmdutil.WritableBinDirname()
	ui.Must(err)

	// If there's an upgrade available and we've made it here, we're meant to
	// install it. Start by downloading it.
	ui.Spinf("downloading <%s>", u.String())
	pathname, err := fileutil.Download(u, fmt.Sprintf(
		"substrate-%s-%s-%s-*.tar.gz",
		v, runtime.GOOS, runtime.GOARCH,
	))
	ui.Must(err)
	defer os.Remove(pathname)
	ui.Stop("ok")

	// Still here, so untar it to overwrite argv[0] with the new binary. Farm
	// this out to tar(1) to avoid buggily reimplementing the distribution.
	ui.Spinf("untarring %s", pathname)
	cmd := exec.Command(
		"tar",
		"-C", dirname,
		"-f", pathname,
		"--strip-components", "2",
		"-x",
		fmt.Sprintf(
			"substrate-%s-%s-%s/bin/substrate",
			v, runtime.GOOS, runtime.GOARCH,
		),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ui.Must(cmd.Run())
	ui.Stop("ok")

	ui.Printf("upgraded Substrate to %s", v)
}
