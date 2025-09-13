package agent

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents [OPTION] ...",
		Short:   "manage agents",
		GroupID: "operations",
	}

	cmd.AddCommand(listCommand())
	cmd.AddCommand(acceptCommand())
	cmd.AddCommand(removeCommand())
	cmd.AddCommand(rejectCommand())
	cmd.AddCommand(healthCommand())

	return cmd
}
