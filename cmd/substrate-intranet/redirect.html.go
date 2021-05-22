package main

// managed by go generate; do not edit by hand

func redirectTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Intranet</title>
<body>
<h1>Intranet</h1>
{{- if .ErrorDescription}}
<p class="error">{{.ErrorDescription}}</p>
<p><a href="{{.Location}}">Try again</a>.</p>
{{- else}}
<p>Redirecting to <a href="{{.Location}}">your identity provider</a>.</p>
{{- end}}
</body>
</html>
`
}
