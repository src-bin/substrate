package upgrade

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

var Domain = "src-bin.com" // change to "src-bin.org" to test in staging or control it in the Makefile if you want to get fancy

func Main(ctx context.Context, cfg *awscfg.Config) {
	no := flag.Bool("no", false, `answer "no" when offered an upgrade; exits 1 if there was an upgrade available`)
	yes := flag.Bool("yes", false, `answer "yes" when offered an upgrade to accept it without confirmation`)
	flag.Usage = func() {
		ui.Print("Usage: substrate upgrade [-no|-yes]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Print the current version for context and to make the fact that we're
	// about to print the prefix make more sense. Then print the URL we'll
	// fetch to see if there's an upgrade available.
	version.Print()
	fromVersion := fmt.Sprintf("%s-%s", version.Version, version.Commit)
	u := &url.URL{
		Scheme: "https",
		Host:   Domain,
		Path: fmt.Sprintf(
			"/substrate/upgrade/%s/%s",
			fromVersion,
			naming.Prefix(),
		),
	}
	ui.Printf("checking <%s> to see if there's a Substrate upgrade available", u.String())

	// Fetch the upgrade URL to see if there's an upgrade available from this
	// version for this customer. Exit 0 if there's not.
	resp, err := http.Get(u.String())
	if err != nil {
		ui.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		ui.Print("there's no Substrate upgrade available yet")
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ui.Fatal(err)
	}
	toVersion := strings.TrimSpace(string(body))

	// Construct the download URL.
	u = &url.URL{
		Scheme: "https",
		Host:   Domain,
		Path: fmt.Sprintf(
			"/substrate-%s-%s-%s.tar.gz",
			toVersion,
			runtime.GOOS,
			runtime.GOARCH,
		),
	}
	ui.Printf("there's a Substrate upgrade available at <%s>", u.String())

	// Exit 1 when there's no upgrade available and either the -no option was
	// given or the user answers "no" to the prompt.
	if *no {
		ui.Print("re-run this command without the -no option to install it")
		os.Exit(1)
	}
	if !*yes {
		if ok, err := ui.Confirmf("upgrade to Substrate %s? (yes/no)", toVersion); err != nil {
			ui.Fatal(err)
		} else if !ok {
			ui.Print("not upgrading Substrate") // not ui.Fatal to suppress the stack trace
			os.Exit(1)
		}
	}

	// Try to preempt common tar(1) failures we encounter when the install
	// directory ownership and permissions.
	pathname, err := os.Executable()
	if err != nil {
		ui.Fatal(err)
	}
	dirname := filepath.Dir(pathname)
	fi, err := os.Stat(dirname)
	if err != nil {
		ui.Fatal(err)
	}
	perm := fi.Mode().Perm()
	sys := fi.Sys().(*syscall.Stat_t)
	if perm&0200 != 0 && sys.Uid == uint32(os.Geteuid()) {
		// writable by owner, which we are
	} else if perm&0020 != 0 && sys.Gid == uint32(os.Getegid()) {
		// writable by owning group, which is our primary group
		// TODO also check supplemental groups if you want to be fancy
	} else if perm&0002 != 0 {
		// writable by anyone, which is bad but not our problem
	} else {
		ui.Printf("%s not writable, so Substrate cannot upgrade itself", dirname)
		os.Exit(1)
	}

	// If there's an upgrade available and we've made it here, we're meant to
	// install it. Start by downloading it.
	ui.Spinf("downloading <%s>", u.String())
	f, err := os.CreateTemp("", fmt.Sprintf(
		"substrate-%s-%s-%s-*.tar.gz",
		toVersion,
		runtime.GOOS,
		runtime.GOARCH,
	))
	if err != nil {
		ui.Fatal(err)
	}
	defer os.Remove(f.Name())
	if resp, err = http.Get(u.String()); err != nil {
		ui.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		ui.Fatalf("<%s> responded %d %s", u.String(), resp.StatusCode, resp.Status)
	}
	if _, err = io.Copy(f, resp.Body); err != nil {
		ui.Fatal(err)
	}
	if err := f.Close(); err != nil {
		ui.Fatal(err)
	}
	ui.Stop("ok")

	// Still here, so untar it to overwrite argv[0] with the new binary. Farm
	// this out to tar(1) to avoid buggily reimplementing the distribution.
	ui.Spinf("untarring %s", f.Name())
	cmd := exec.Command(
		"tar",
		"-C", dirname,
		"-f", f.Name(),
		"-x",
		"--strip-components", "2",
		fmt.Sprintf(
			"substrate-%s-%s-%s/bin/substrate",
			toVersion,
			runtime.GOOS,
			runtime.GOARCH,
		),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Fatal(err)
	}
	ui.Stop("ok")

	ui.Printf("upgraded Substrate to %s", toVersion)
}
