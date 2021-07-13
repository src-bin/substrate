package main

// managed by go generate; do not edit by hand

func accountsTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Accounts</title>
<body>
<h1>Accounts</h1>
<p class="context">Here are all the AWS accounts in your organization. Once you've logged into the AWS Console using your identity provider, use this table to assume roles in all your accounts in the AWS Console. If you need command-line access, use <kbd>eval $(substrate-credentials)</kbd>, the <a href="credential-factory">Credential Factory</a>, or the <a href="instance-factory">Instance Factory</a>.</p>
<h2>Special accounts</h2>
<table border="1" cellpadding="2" cellspacing="2">
<tr>
    <th nowrap>Name</th>
    <th nowrap>Account Number</th>
    <th colspan="2"nowrap>Launch the AWS Console as...</th>
</tr>
<tr>
    <td>management</td>
    <td>{{.ManagementAccount.Id}}</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.ManagementAccount.Id}}&displayName=OrganizationAdministrator&roleName=OrganizationAdministrator" target="_blank">OrganizationAdministrator</a></td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.ManagementAccount.Id}}&displayName=OrganizationReader&roleName=OrganizationReader" target="_blank">OrganizationReader</a></td>
</tr>
<tr>
    <td>{{.AuditAccount.Name}}</td>
    <td>{{.AuditAccount.Id}}</td>
    <td>&nbsp;</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.AuditAccount.Id}}&displayName={{.AuditAccount.Name}}+Auditor&roleName=Auditor" target="_blank">Auditor</a></td>
</tr>
<tr>
    <td>{{.DeployAccount.Name}}</td>
    <td>{{.DeployAccount.Id}}</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.DeployAccount.Id}}&displayName=DeployAdministrator&roleName=DeployAdministrator" target="_blank">DeployAdministrator</a></td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.DeployAccount.Id}}&displayName={{.DeployAccount.Name}}+Auditor&roleName=Auditor" target="_blank">Auditor</a></td>
</tr>
<tr>
    <td>{{.NetworkAccount.Name}}</td>
    <td>{{.NetworkAccount.Id}}</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.NetworkAccount.Id}}&displayName=NetworkAdministrator&roleName=NetworkAdministrator" target="_blank">NetworkAdministrator</a></td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.NetworkAccount.Id}}&displayName={{.NetworkAccount.Name}}+Auditor&roleName=Auditor" target="_blank">Auditor</a></td>
</tr>
</table>
<h2>Service accounts</h2>
<table border="1" cellpadding="2" cellspacing="2">
<tr>
    <th nowrap>Domain</th>
    <th nowrap>Environment</th>
    <th nowrap>Quality</th>
    <th nowrap>Account Number</th>
    <th colspan="2"nowrap>Launch the AWS Console as...</th>
</tr>
{{- range .ServiceAccounts}}
<tr>
    <td>{{.Tags.Domain}}</td>
    <td>{{.Tags.Environment}}</td>
    <td>{{.Tags.Quality}}</td>
    <td>{{.Id}}</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.Id}}&displayName={{.Tags.Domain}}+{{.Tags.Environment}}+{{.Tags.Quality}}+Administrator&roleName=Administrator" target="_blank">Administrator</a></td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.Id}}&displayName={{.Tags.Domain}}+{{.Tags.Environment}}+{{.Tags.Quality}}+Auditor&roleName=Auditor" target="_blank">Auditor</a></td>
</tr>
{{- end}}
</table>
<h2>Admin accounts</h2>
<table border="1" cellpadding="2" cellspacing="2">
<tr>
    <th nowrap>Quality</th>
    <th nowrap>Account Number</th>
    <th colspan="2"nowrap>Launch the AWS Console as...</th>
</tr>
{{- range .AdminAccounts}}
<tr>
    <td>{{.Tags.Quality}}</td>
    <td>{{.Id}}</td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.Id}}&displayName=admin+{{.Tags.Quality}}+Administrator&roleName=Administrator" target="_blank">Administrator</a></td>
    <td><a href="https://signin.aws.amazon.com/switchrole?account={{.Id}}&displayName=admin+{{.Tags.Quality}}+Auditor&roleName=Auditor" target="_blank">Auditor</a></td>
</tr>
{{- end}}
</table>
</body>
</html>
`
}
