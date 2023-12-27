package role

import (
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/role/create"
	"github.com/src-bin/substrate/cmd/substrate/role/delete"
	"github.com/src-bin/substrate/cmd/substrate/role/list"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role create|delete|list",
		Short: "TODO role.Command().Short",
		Long:  `TODO role.Command().Long`,
	}

	cmd.AddCommand(create.Command())
	cmd.AddCommand(delete.Command())
	cmd.AddCommand(list.Command())

	return cmd
}
