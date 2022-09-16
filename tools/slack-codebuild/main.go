package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/src-bin/substrate/fileutil"
)

const codebuildLogMaxLen = 2048 // maximum Slack message length is theoretically 40,000 characters but this seems to be just about all I can get to fit in a single message

func codebuildLog(ctx context.Context) string {
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
	s := b.String()
	if len(s) < codebuildLogMaxLen {
		return s
	}
	return s[len(s)-codebuildLogMaxLen:]
}

func main() {
	ctx := context.Background()

	buildURL := (&url.URL{
		Scheme: "https",
		Host:   "src-bin.net",
		Path:   "/accounts",
		RawQuery: url.Values{
			"next":   []string{os.Getenv("CODEBUILD_BUILD_URL")},
			"number": []string{"412086678291"},
			"role":   []string{"Auditor"},
		}.Encode(),
	}).String()

	// If the build's failing, report it to Slack.
	if os.Getenv("CODEBUILD_BUILD_SUCCEEDING") != "1" {
		slack(fmt.Sprintf(
			"Substrate build has failed!\n"+
				"\n"+
				"Source tree: https://github.com/src-bin/substrate/tree/%s\n"+
				"\n"+
				"Build log: %s\n"+
				"\n"+
				"```\n%s```\n",
			os.Getenv("CODEBUILD_RESOLVED_SOURCE_VERSION"),
			buildURL,
			codebuildLog(ctx),
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

	// Send the release announcement to be shared with customers (for tagged
	// releases) or just noted as a successful build (for untagged releases).
	content, err := fileutil.ReadFile("substrate.version")
	if err != nil {
		log.Fatal(err)
	}
	version := strings.Trim(string(content), "\r\n")
	out, err = exec.Command("git", "show", "--format=%h", "--no-patch").Output()
	if err != nil {
		log.Fatal(err)
	}
	commit := strings.Trim(string(out), "\r\n")
	var filenames []string
	for goOS := range []string{"darwin", "linux"} {
		for goArch := range []string{"amd64", "arm64"} {
			filenames = append(filenames, fmt.Sprintf(
				"substrate-%s-%s-%s-%s.tar.gz",
				version, commit, goOS, goArch,
			))
		}
	}
	var tmpl *template.Template
	if taggedRelease {
		tmpl, err = template.New("release").Parse(
			"Substrate {{.Version}} is out!\n" +
				"\n" +
				"Full release notes: https://src-bin.com/substrate/manual/releases/#{{.Version}}\n" +
				"\n" +
				"Get it by running `substrate upgrade` or downloading the appropriate release tarball:\n" +
				"{{range .Filenames -}}\n" +
				"https://src-bin.com/{{.}}\n" +
				"{{end -}}\n",
		)
	} else {
		tmpl, err = template.New("release").Parse(
			"Substrate {{.Version}} (an untagged release) built successfully\n" +
				"\n" +
				"Downloads:\n" +
				"{{range .Filenames -}}\n" +
				"https://src-bin.com/{{.}}\n" +
				"{{end -}}\n" +
				"\n" +
				"These tarballs will be deleted on the next `make -C release clean`\n",
		)
	}
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
	if !taggedRelease {
		text += fmt.Sprintf(
			"\n"+
				"Source tree: https://github.com/src-bin/substrate/tree/%s\n"+
				"\n"+
				"Build log: %s\n",
			os.Getenv("CODEBUILD_RESOLVED_SOURCE_VERSION"),
			buildURL,
		)
	}
	slack(text)

	// Don't bother sending the release checklist as individual Slack
	// messages for untagged releases.
	if !taggedRelease {
		return
	}

	// Send a reminder to deploy the website.
	slack("Deploy release notes and documentation updates to https://src-bin.com/substrate/manual/.")

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
