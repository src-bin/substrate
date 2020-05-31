package main


import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func main() {
	domain := aws.String("admin") // to match the interface of flag.String
	environment := aws.String("admin") // to match the interface of flag.String
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()

	lines, err := ui.EditFile(
		OktaMetadataFilename,
		"here is your current identity provider metadata XML:",
		"paste your identity provider metadata XML from Okta",
	)
	if err != nil {
		log.Fatal(err)
	}
	metadata := strings.Join(lines, "\n") + "\n" // ui.EditFile is line-oriented but this instance isn't

	sess :=

	networkAccount, err := awsorgs.EnsureAccount(
		svc,
		accounts.Network,
		awsorgs.EmailForAccount(org, accounts.Network),
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", networkAccount.Id)
	//log.Printf("%+v", networkAccount)

}
