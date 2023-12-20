package account

import (
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/account/list"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account adopt|create|list",
		Short: "TODO account.Command().Short",
		Long:  `TODO account.Command().Long`,
	}

	// TODO adopt
	// TODO create
	cmd.AddCommand(list.Command())

	return cmd
}
