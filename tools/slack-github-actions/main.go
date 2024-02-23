package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func main() {

	// Figure out if this is a tagged release or not. We can skip a bunch of
	// Slack messages for untagged releases to be generally less spammy.
	var taggedRelease bool
	if err := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD").Run(); err == nil {
		taggedRelease = true
	} else {
		if _, ok := err.(*exec.ExitError); !ok {
			log.Fatal(err)
		}
		taggedRelease = false
	}

	// Gather all the raw materials for formatting release announcements.
	content, err := os.ReadFile("substrate.version")
	if err != nil {
		log.Fatal(err)
	}
	version := strings.TrimSpace(string(content))
	var filenames []string
	for _, goOS := range []string{"darwin", "linux"} {
		for _, goArch := range []string{"amd64", "arm64"} {
			filenames = append(filenames, fmt.Sprintf(
				"substrate-%s-%s-%s.tar.gz",
				version, goOS, goArch,
			))
		}
	}

	// Construct the main Slack message, which for a tagged release is going
	// to be sent several times.
	var text string
	if taggedRelease {
		text = parseAndExecuteTemplate(
			"Substrate {{.Version}} is out!\n"+
				"\n"+
				"Full release notes: https://docs.substrate.tools/substrate/releases#{{.Version}}\n"+
				"\n"+
				"Get it by running `substrate upgrade` or downloading the appropriate release tarball:\n"+
				"{{range .Filenames -}}\n"+
				"https://src-bin.com/{{.}}\n"+
				"{{end -}}\n",
			struct {
				Filenames []string
				Version   string
			}{filenames, version},
		)
	} else {
		text = parseAndExecuteTemplate(
			"Substrate {{.Version}} (an untagged release) built successfully\n"+
				"\n"+
				"Downloads:\n"+
				"{{range .Filenames -}}\n"+
				"https://src-bin.com/{{.}}\n"+
				"{{end -}}\n"+
				"\n"+
				"These tarballs will be deleted on the next `make -C release clean`\n"+
				"\n"+
				"<{{.ServerURL}}/{{.Repo}}/actions/runs/{{.RunId}}|GitHub Actions log> / <{{.ServerURL}}/{{.Repo}}/tree/{{.SHA}}|source tree>",
			struct {
				Filenames                   []string
				Version                     string
				ServerURL, Repo, RunId, SHA string
			}{
				filenames,
				version,
				os.Getenv("GITHUB_SERVER_URL"),
				os.Getenv("GITHUB_REPOSITORY"),
				os.Getenv("GITHUB_RUN_ID"),
				os.Getenv("GITHUB_SHA"),
			},
		)
	}

	// Announce the release builds, whether tagged or untagged, in #substrate
	// (or whichever channel is this webhook's default).
	slack(map[string]string{"text": text})

	// For tagged builds, announce the release directly to customer channels
	// found in the environment.
	if taggedRelease {
		for _, channel := range split(os.Getenv("CHANNELS")) {
			slack(map[string]string{"channel": strings.TrimSpace(channel), "text": text})
		}
	}

	// TODO For tagged builds, announce the release via the mailing list, too.

}

func parseAndExecuteTemplate(t string, i interface{}) string {
	tmpl, err := template.New("tmpl").Parse(t)
	if err != nil {
		log.Fatal(err)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, i); err != nil {
		log.Fatal(err)
	}
	return b.String()
}

func slack(body map[string]string) {
	b, err := json.MarshalIndent(body, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("slack webhook\n request body:  %s\n", string(b))
	if slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL"); slackWebhookURL != "" {
		resp, err := http.Post(slackWebhookURL, "application/json; charset=utf-8", bytes.NewBuffer(b))
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		if b, err = io.ReadAll(resp.Body); err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" response body: %s\n", string(b))
	} else {
		fmt.Print(" not sent because SLACK_WEBHOOK_URL isn't in the environment\n")
	}
}

func split(s string) []string {
	if s == "" {
		return []string{}
	}
	ss := strings.Split(s, ",")
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
	}
	return ss
}
