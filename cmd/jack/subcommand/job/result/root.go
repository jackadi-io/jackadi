package result

import "github.com/spf13/cobra"

func ResultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "results [OPTION] ...",
		Short:   "manage results",
		GroupID: "operations",
	}

	cmd.AddCommand(getCommand())
	cmd.AddCommand(listCommand())

	return cmd
}
