package main

// managed by go generate; do not edit by hand

func instanceTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Instance Factory</title>
<body>
<h1>Instance Factory</h1>
{{- if .Error}}
<p class="error">{{.Error}}</p>
{{- end}}
<p>Provisioning a <strong>{{.InstanceType}}</strong> instance in <strong>{{.Region}}</strong>.</p>
<p><strong>TODO actually invoke <code>ec2:RunInstances</code> (nothing has happened).</strong></p>
<p><a href="instance-factory">See all your instances</a></p>
<hr>
<pre>{{.Debug}}</pre>
</body>
</html>
`
}