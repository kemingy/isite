package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg/tools"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "remove expired caches and generated output",
	RunE: func(cmd *cobra.Command, _ []string) error {
		removedCaches, err := tools.PruneThemeCache()
		if err != nil {
			return err
		}
		fmt.Printf("removed %d expired theme cache(s)\n", removedCaches)
		if !cmd.Flags().Changed("output") && !cmd.Root().PersistentFlags().Changed("output") {
			return nil
		}
		removedOutput, err := tools.PruneOutput(outputDir)
		if err != nil {
			return err
		}
		if removedOutput {
			fmt.Printf("removed generated output %s\n", outputDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
