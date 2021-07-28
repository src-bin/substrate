package main

// managed by go generate; do not edit by hand

func credentialFactoryAuthorizeTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Credential Factory</title>
<body>
<h1>Credential Factory</h1>
<p class="context">This tool authorizes <code>substrate-credentials</code> to fetch short-lived credentials that administrators can use to work safely from their laptops. Alternatively, the <a href="credential-factory">Credential Factory</a> mints short-lived credentials that can be copied into a shell and the <a href="instance-factory">Instance Factory</a> provisions EC2 instances. All three strategies for minting credentials confer the same privileges.</p>
<p>The <code>substrate-credentials</code> invocation with token <strong>{{.}}</strong> is now authorized. You may close this browser window and return to your terminal now.</p>
</body>
</html>
`
}
