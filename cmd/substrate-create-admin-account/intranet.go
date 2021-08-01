package main

import _ "embed"

//go:generate env GOARCH=amd64 GOOS=linux go build ../substrate-intranet
//go:generate zip substrate-intranet.zip substrate-intranet
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte
