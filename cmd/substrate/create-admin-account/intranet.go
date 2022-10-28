package createadminaccount

import _ "embed"

//go:generate env GOARCH=arm64 GOOS=linux go build -o bootstrap ../../substrate-intranet
//go:generate zip substrate-intranet.zip bootstrap
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte
