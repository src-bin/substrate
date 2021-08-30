package main

// managed by go generate; do not edit by hand

func instanceFactoryKeyPairTemplate() string {
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
<p>SSH public key for <strong>{{.PrincipalId}}</strong>:</p>
<textarea cols="80" name="public_key_material" rows="3"></textarea>
<p><input type="submit" value="Import public key"></p>
<input name="region" type="hidden" value="{{.Region}}">
</form>
<p>Or <a href="instance-factory">cancel</a></p>
</body>
</html>
`
}