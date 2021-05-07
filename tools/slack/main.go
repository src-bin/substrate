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

Full release notes: https://src-bin.co/substrate/manual/releases/#{{.Version}}

Downloads:
{{range .Filenames -}}
https://src-bin.co/{{.}}
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
