package versionutil

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

var Domain = "src-bin.com" // change to "src-bin.org" to test in staging or control it in the Makefile if you want to get fancy

func CheckForUpgrade() (v string, ok bool, err error) {
	u := UpgradeURL()
	var resp *http.Response
	if resp, err = http.Get(u.String()); err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return
	}
	ui.Must(err)
	v = strings.TrimSpace(string(body))
	ok = true
	return
}

func DownloadURL(v, goOS, goArch string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   Domain,
		Path: fmt.Sprintf(
			"/substrate-%s-%s-%s.tar.gz",
			v, goOS, goArch,
		),
	}
}

func UpgradeURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   Domain,
		Path:   fmt.Sprintf("/substrate/upgrade/%s", version.Version),
	}
}
