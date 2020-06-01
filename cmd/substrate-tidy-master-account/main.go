package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/roles"
)

func main() {

	sess := session.Must(awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{}))

	if err := awsiam.DeleteAllAccessKeys(
		iam.New(sess),
		roles.OrganizationAdministrator,
	); err != nil {
		log.Fatal(err)
	}

}
