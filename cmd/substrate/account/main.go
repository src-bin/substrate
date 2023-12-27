package account

import (
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/account/adopt"
	"github.com/src-bin/substrate/cmd/substrate/account/create"
	"github.com/src-bin/substrate/cmd/substrate/account/list"
	"github.com/src-bin/substrate/cmd/substrate/account/update"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account adopt|create|list|update",
		Short: "TODO account.Command().Short",
		Long:  `TODO account.Command().Long`,
	}

	cmd.AddCommand(adopt.Command())
	cmd.AddCommand(create.Command())
	cmd.AddCommand(list.Command())
	cmd.AddCommand(update.Command())
	// TODO cmd.AddCommand(delete.Command()) ???

	return cmd
}
