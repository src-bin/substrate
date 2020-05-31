package main

import (
	"log"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/roles"
)

func main() {

	sess, err := awssessions.Master(roles.OrganizationAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := awsiam.DeleteAllAccessKeys(
		iam.New(svc),
		roles.OrganizationAdministrator,
	); err != nil {
		log.Fatal(err)
	}

}
