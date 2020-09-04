package main

// managed by go generate; do not edit by hand

func indexTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Instance Factory</title>
<body>
<h1>Instance Factory</h1>
<p class="context">This tool provisions EC2 instances that administrators can use to work in the cloud. Alternatively, the <a href="credential-factory">Credential Factory</a> mints short-lived credentials with the same privileges as these EC2 instances that administrators can use (more) safely on their laptops.</p>
{{- if .Error}}
<p class="error">{{.Error}}</p>
{{- end}}
<form method="GET">
<p>Provision a new EC2 instance in:
{{- range $i, $region := .Regions}}
<input name="region" type="submit" value="{{$region}}">
{{- end}}
</p>
</form>
<table border="1" cellpadding="2" cellspacing="2">
<tr>
    <th>Hostname</th>
    <th>Availability Zone</th>
    <th>Instance Type</th>
    <th>Provision Time</th>
    <th>&nbsp;</th>
</tr>
{{- range $i, $instance := .Instances}}
<tr>
    <td>{{$instance.PublicDnsName}}</td>
    <td>{{$instance.Placement.AvailabilityZone}}</td>
    <td>{{$instance.InstanceType}}</td>
    <td>{{$instance.LaunchTime}}</td>
    <td><input name="terminate" type="submit" value="{{$instance.InstanceId}}"></td>
</tr>
{{- end}}
</table>
<hr>
<pre>{{.Debug}}</pre>
</body>
</html>
`
}
