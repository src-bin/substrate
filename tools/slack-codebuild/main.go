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

	// If the build's failing, report it to Slack.
	if os.Getenv("CODEBUILD_BUILD_SUCCEEDING") != "1" {
		slack(fmt.Sprintf(
			"Substrate build %s of https://github.com/src-bin/substrate/tree/%s failed!\n%s",
			os.Getenv("CODEBUILD_BUILD_NUMBER"),
			os.Getenv("CODEBUILD_RESOLVED_SOURCE_VERSION"),
			os.Getenv("CODEBUILD_BUILD_URL"),
		))
		return
	}

	// If it's succeeding but it's still early, sit tight.
	if true {
		return
	}

	// But if the build's succeeding and this is the end, announce it.

	// Send the release announcement to be shared with customers.
	out, err := exec.Command("make", "release-filenames").Output()
	if err != nil {
		log.Fatal(err)
	}
	filenames := strings.Split(strings.Trim(string(out), "\r\n"), "\n")
	out, err = exec.Command("make", "release-version").Output()
	if err != nil {
		log.Fatal(err)
	}
	version := strings.Trim(string(out), "\r\n")
	tmpl, err := template.New("release").Parse(
		`Substrate {{.Version}} is out!

Full release notes: https://src-bin.com/substrate/manual/releases/#{{.Version}}

Downloads:
{{range .Filenames -}}
https://src-bin.com/{{.}}
{{end -}}`,
	)
	if err != nil {
		log.Fatal(err)
	}
	var b strings.Builder
	err = tmpl.Execute(&b, struct {
		Filenames []string
		Version   string
	}{filenames, version})
	if err != nil {
		log.Fatal(err)
	}
	text := b.String()
	slack(text)
	slack("Push release notes and documentation updates to https://src-bin.com/substrate/manual/.")

	// Send the checklist of customers who need the announcement.
	for _, customer := range split(os.Getenv("CUSTOMERS_ANNOUNCE")) {
		slack(fmt.Sprintf("Share or copy/paste the release announcement to *%s*", customer))
	}

	// Send the checklist of customers I'm supposed to upgrade myself.
	for _, customer := range split(os.Getenv("CUSTOMERS_UPGRADE")) {
		slack(fmt.Sprintf("Upgrade Substrate for *%s*", customer))
	}

	// Send the checklist of customers for whom I have other obligations.
	for _, customer := range split(os.Getenv("CUSTOMERS_PIN")) { // not alphabetical but these happen after upgrades
		slack(fmt.Sprintf("Pin the new version of Substrate for *%s*", customer))
	}

}

func slack(text string) {
	body, err := json.MarshalIndent(map[string]string{"text": text}, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
	if slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL"); slackWebhookURL != "" {
		resp, err := http.Post(slackWebhookURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(body))
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
