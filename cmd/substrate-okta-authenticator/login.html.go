package main

// managed by go generate; do not edit by hand

func loginTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Intranet</title>
<body>
<h1>Intranet</h1>
<p>Hello, <a href="mailto:{{.AccessToken.Subject}}">{{.AccessToken.Subject}}</a>!</p>
{{- if .Location}}
<p>Redirecting to <a href="{{.Location}}">{{.Location}}</a>.</p>
{{- end}}
</body>
</html>
`
}
