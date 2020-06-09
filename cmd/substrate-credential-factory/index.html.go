package main

// managed by go generate; do not edit by hand

func indexTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Credential Factory</title>
<body>
<h1>Credential Factory</h1>
<p class="context">This tool mints short-lived credentials that administrators can use to work safely from their laptops. Alternatively, the <a href="instance-factory">Instance Factory</a> provisions EC2 instances with the same privileges as these credentials that administrators can use to work in the cloud.</p>
{{- if .Error}}
<p class="error">{{.Error}}</p>
{{- end}}
<form method="POST">
<p><input type="submit" value="Mint new AWS credentials"> which will expire in 12 hours</p>
</form>
<hr>
<pre>{{.Debug}}</pre>
</body>
</html>
`
}
