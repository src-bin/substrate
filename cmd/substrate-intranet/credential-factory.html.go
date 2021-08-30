package main

// managed by go generate; do not edit by hand

func credentialFactoryTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Credential Factory</title>
<body>
<h1>Credential Factory</h1>
<p class="context">This tool mints short-lived credentials that administrators can use to work safely from their laptops. Alternatively, the <a href="instance-factory">Instance Factory</a> provisions EC2 instances with the same privileges as these credentials that administrators can use to work in the cloud.</p>
<p>Paste this into a shell to set environment variables (taking care to preserve the leading space to keep it out of your <code>~/.bash_history</code>):</p>
<p><big><kbd>&nbsp;export AWS_ACCESS_KEY_ID="{{.AccessKeyId}}" AWS_SECRET_ACCESS_KEY="{{.SecretAccessKey}}" AWS_SESSION_TOKEN="{{.SessionToken}}"</kbd></big></p>
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
<form method="GET">
<p><input type="submit" value="Mint new AWS credentials"> which will expire in 12 hours</p>
</form>
</body>
</html>
`
}