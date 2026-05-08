package node

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nodes [OPTION] ...",
		Short:   "manage nodes",
		GroupID: "operations",
	}

	cmd.AddCommand(listCommand())
	cmd.AddCommand(acceptCommand())
	cmd.AddCommand(removeCommand())
	cmd.AddCommand(rejectCommand())
	cmd.AddCommand(healthCommand())

	return cmd
}
