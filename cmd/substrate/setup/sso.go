package setup

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssso"
	"github.com/src-bin/substrate/ui"
)

func sso(ctx context.Context, mgmtCfg *awscfg.Config) bool {

	instances, err := awssso.ListInstances(ctx, mgmtCfg)
	ui.Must(err)
	ui.Debug(instances) // XXX
	if len(instances) > 0 {
		// TODO ask if they want Substrate to manage IAM Identity Center and return if they don't
	} else {
		return false // not managing IAM Identity Center; tell them how to start managing it
	}

	accounts, err := mgmtCfg.ListAccounts(ctx)
	ui.Must(err)

	for _, instance := range instances {
		permissionSets, err := awssso.ListPermissionSets(ctx, mgmtCfg, instance)
		ui.Must(err)
		ui.Debug(permissionSets) // XXX
		for _, permissionSet := range permissionSets {
			for _, account := range accounts {
				assignments, err := awssso.ListAccountAssignments(ctx, mgmtCfg, instance, permissionSet, aws.ToString(account.Id))
				ui.Must(err)
				ui.Debug(assignments) // XXX
			}
		}
	}

	substrateAccount, err := mgmtCfg.FindSubstrateAccount(ctx)
	ui.Must(err)
	ui.Must(awsorgs.RegisterDelegatedAdministrator(
		ctx,
		mgmtCfg,
		aws.ToString(substrateAccount.Id),
		"sso.amazonaws.com",
	))

	return true // managing IAM Identity Center
}
