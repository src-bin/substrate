package role

import (
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/role/create"
	"github.com/src-bin/substrate/cmd/substrate/role/delete"
	"github.com/src-bin/substrate/cmd/substrate/role/list"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "manage AWS IAM roles",
	}

	cmd.AddCommand(create.Command())
	cmd.AddCommand(delete.Command())
	cmd.AddCommand(list.Command())

	return cmd
}
