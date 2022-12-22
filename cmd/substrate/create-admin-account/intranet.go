package createadminaccount

import _ "embed"

//go:generate env GOARCH=arm64 GOOS=linux go build -o bootstrap ../../substrate-intranet
//go:generate touch -t 202006100000.00 bootstrap
//go:generate zip -X substrate-intranet.zip bootstrap
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte
