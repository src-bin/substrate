package main

// managed by go generate; do not edit by hand

func indexTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Intranet</title>
<body>
<h1>Intranet</h1>
<pre>{{.Debug}}</pre>
</body>
</html>
`
}
