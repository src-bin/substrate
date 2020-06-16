package main

// managed by go generate; do not edit by hand

func credentialsTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Credential Factory</title>
<body>
<h1>Credential Factory</h1>
{{- if .Error}}
<p class="error">{{.Error}}</p>
{{- end}}
{{- with .Credentials}}
<table border="1" cellpadding="2" cellspacing="2">
<tr>
    <th nowrap>Access Key ID</th>
    <td>{{.AccessKeyId}}</td>
</tr>
<tr>
    <th nowrap>Secret Access Key</th>
    <td>{{.SecretAccessKey}}</td>
</tr>
<tr>
    <th nowrap>Session Token</th>
    <td>{{.SessionToken}}</td>
</tr>
<tr>
    <th nowrap>Expiration</th>
    <td>{{.Expiration}}</td>
</tr>
</table>
<p>Or, paste this into a shell to set environment variables (taking care to preserve the leading space):</p>
<p><kbd>&nbsp;export AWS_ACCESS_KEY_ID="{{.AccessKeyId}}" AWS_SECRET_ACCESS_KEY="{{.SecretAccessKey}}" AWS_SESSION_TOKEN="{{.SessionToken}}"</kbd></p>
{{- end}}
<form method="POST">
<p>OK, I have them, <a href="credential-factory">go back</a>, or <input type="submit" value="Mint new AWS credentials"> which will expire in one hour</p>
</form>
<hr>
<pre>{{.Debug}}</pre>
</body>
</html>
`
}
