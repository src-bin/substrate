package main

// managed by go generate; do not edit by hand

func instanceTypeTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Instance Factory</title>
<body>
<h1>Instance Factory</h1>
{{- if .Error}}
<p class="error">{{.Error}}</p>
{{- end}}
<form action="instance-factory" method="POST">
<p>Provisioning in <strong>{{.Region}}</strong>.</p>
<p>Choose your instance type:</p>
<table>
{{- range $instanceFamily, $instanceTypes := .InstanceFamilies}}
<tr>
{{- range $i, $instanceType := $instanceTypes}}
<td><input name="instance_type" type="submit" value="{{$instanceType}}"></td>
{{- end}}
</tr>
{{- end}}
</table>
<input name="region" type="hidden" value="{{.Region}}">
</form>
<p>Or <a href="instance-factory">cancel</a></p>
</body>
</html>
`
}
