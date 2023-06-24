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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/src-bin/substrate/fileutil"
)

const codebuildLogMaxLen = 1500 // maximum Slack message length is theoretically 40,000 characters but empirically it seems a lot lower to avoid splitting

func codebuildLog(ctx context.Context) string {

	time.Sleep(10 * time.Second) // give CloudWatch Logs time to catch up with reality

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
	if i := strings.Index(s, "FAIL"); i != -1 {
		if i < codebuildLogMaxLen { // a prefix of the log contains a failure
			return s[:codebuildLogMaxLen]
		}
		if len(s) >= i+codebuildLogMaxLen/4 { // a middle chunk contains a failure
			return s[i-3*codebuildLogMaxLen/4 : i+codebuildLogMaxLen/4]
		}
	}
	return s[len(s)-codebuildLogMaxLen:] // the end either contains a failure or is otherwise most likely to be interesting
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
			"role":   []string{"Administrator"},
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
	out, err := exec.Command("git", "show", "--format=%h", "--no-patch").Output()
	if err != nil {
		log.Fatal(err)
	}
	commit := strings.Trim(string(out), "\r\n")
	var filenames, trialFilenames []string
	for _, goOS := range []string{"darwin", "linux"} {
		for _, goArch := range []string{"amd64", "arm64"} {
			filenames = append(filenames, fmt.Sprintf(
				"substrate-%s-%s-%s-%s.tar.gz",
				version, commit, goOS, goArch,
			))
			trialFilenames = append(trialFilenames, fmt.Sprintf(
				"substrate-%s-%s-%s-%s.tar.gz",
				version, "trial", goOS, goArch,
			))
		}
	}
	var text string
	if taggedRelease {
		text, err = parseAndExecuteTemplate(
			"Substrate {{.Version}} is out!\n"+
				"\n"+
				"Full release notes: https://docs.src-bin.com/substrate/releases#{{.Version}}\n"+
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
		text, err = parseAndExecuteTemplate(
			"Substrate {{.Version}} (an untagged release) built successfully\n"+
				"\n"+
				"Downloads:\n"+
				"{{range .Filenames -}}\n"+
				"https://src-bin.com/{{.}}\n"+
				"{{end -}}\n"+
				"\n"+
				"These tarballs will be deleted on the next `make -C release clean`\n",
			struct {
				Filenames []string
				Version   string
			}{filenames, version},
		)
	}
	if err != nil {
		log.Fatal(err)
	}
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
	text, err = parseAndExecuteTemplate(
		"Trial Substrate {{.Version}}:\n"+
			"{{range .TrialFilenames -}}\n"+
			"https://src-bin.com/{{.}}\n"+
			"{{end -}}\n",
		struct {
			TrialFilenames []string
			Version        string
		}{trialFilenames, version},
	)
	if err != nil {
		log.Fatal(err)
	}
	slack(text)

	// Don't bother sending the release checklist as individual Slack
	// messages for untagged releases.
	if !taggedRelease {
		return
	}

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

func parseAndExecuteTemplate(t string, i interface{}) (string, error) {
	tmpl, err := template.New("tmpl").Parse(t)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, i); err != nil {
		return "", err
	}
	return b.String(), nil
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
