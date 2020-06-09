package lambdautil

// managed by go generate; do not edit by hand

func errorTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Error</title>
<body>
<h1>Error</h1>
<p class="error">{{.}}</p>
</body>
</html>
`
}
