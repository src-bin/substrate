package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func main() {
	ctx := context.Background()

	// If the build's failing, report it to Slack.
	if os.Getenv("CODEBUILD_BUILD_SUCCEEDING") != "1" {

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatal(err)
		}
		out, err := cloudwatchlogs.NewFromConfig(cfg).GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String("CodeBuild/substrate"), // to match src-bin/modules/build/regional
			LogStreamName: aws.String(os.Getenv("CODEBUILD_LOG_PATH")),
		})
		if err != nil {
			log.Fatal(err)
		}
		var b strings.Builder
		for _, e := range out.Events {
			b.WriteString(aws.ToString(e.Message))
		}

		slack(fmt.Sprintf(
			"Substrate build %s of https://github.com/src-bin/substrate/tree/%s failed!\n%s\n\n```\n%s```",
			os.Getenv("CODEBUILD_BUILD_NUMBER"),
			os.Getenv("CODEBUILD_RESOLVED_SOURCE_VERSION"),
			os.Getenv("CODEBUILD_BUILD_URL"),
			b.String(),
		))
		return
	}

	// If it's succeeding but it's still early, sit tight. Yes, "pre_build" is
	// our last build step, because it's simplest to keep things linear and the
	// "pre_build" phase will skip uploading assets when it fails, which is how
	// we get our only-actually-release-tagged-builds behavior.
	if os.Args[1] != "pre_build" {
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
		"Substrate {{.Version}} is out!\n" +
			"\n" +
			"Full release notes: https://src-bin.com/substrate/manual/releases/#{{.Version}}\n" +
			"\n" +
			"Get it by running `substrate upgrade` or downloading the appropriate release tarball:\n" +
			"{{range .Filenames -}}\n" +
			"https://src-bin.com/{{.}}\n" +
			"{{end -}}", // no trailing newline in Slack messages
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
