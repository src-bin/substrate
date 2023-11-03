package terraform

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	no := flag.Bool("no", false, `answer "no" when offered an upgrade; exits 1 if there was an upgrade available`)
	yes := flag.Bool("yes", false, `answer "yes" when offered an upgrade to accept it without confirmation`)
	flag.Usage = func() {
		ui.Print("Usage: substrate terraform [-no|-yes]")
		flag.PrintDefaults()
	}
	flag.Parse()

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	ui.Printf("Substrate requires Terraform version %s", terraform.RequiredVersion())
	v, err := terraform.InstalledVersion()
	if err != nil {
		ui.Printf(
			"couldn't determine what version Terraform is installed (%v) so installing version %s",
			err,
			terraform.RequiredVersion(),
		)
	} else if v == terraform.RequiredVersion() {
		ui.Printf("Terraform is already at version %s", v)
		return
	}

	// Construct the download URL.
	u := &url.URL{
		Scheme: "https",
		Host:   "releases.hashicorp.com",
		Path: fmt.Sprintf(
			"/terraform/%s/terraform_%s_%s_%s.zip",
			terraform.RequiredVersion(),
			terraform.RequiredVersion(),
			runtime.GOOS,
			runtime.GOARCH,
		),
	}
	ui.Printf("it's available at <%s>", u.String())

	// Be pretty sure the directory where substrate is stored is writable.
	dirname, err := cmdutil.WritableBinDirname()
	ui.Must(err)

	// Exit 1 when Terraform needs upgrading and either the -no option was
	// given or the user answers "no" to the prompt.
	if *no {
		ui.Print("re-run this command without the -no option to install it")
		os.Exit(1)
	}
	if !*yes {
		if ok, err := ui.Confirmf("upgrade to Terraform %s? (yes/no)", terraform.RequiredVersion()); err != nil {
			ui.Fatal(err)
		} else if !ok {
			ui.Print("not upgrading Terraform") // not ui.Fatal to suppress the stack trace
			os.Exit(1)
		}
	}

	// If there's an upgrade available and we've made it here, we're meant to
	// install it. Start by downloading it.
	ui.Spinf("downloading <%s>", u.String())
	pathname, err := fileutil.Download(u, fmt.Sprintf(
		"terraform-%s-%s-%s-*.zip",
		terraform.RequiredVersion(),
		runtime.GOOS,
		runtime.GOARCH,
	))
	ui.Must(err)
	defer os.Remove(pathname)
	ui.Stop("ok")

	// Still here, so unzip it alongside this binary. Farm this out to
	// unzip(1) because it's pretty good at this.
	ui.Spinf("unzipping %s", pathname)
	cmd := exec.Command(
		"unzip",
		"-d", dirname,
		"-o",
		"-qq",
		pathname,
		"terraform",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ui.Must(cmd.Run())
	ui.Stop("ok")

}

// Synopsis returns a one-line, short synopsis of the command.
func Synopsis() string {
	return "ensures Terraform is installed"
}
