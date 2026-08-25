package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg/tools"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "remove expired theme caches",
	RunE: func(*cobra.Command, []string) error {
		removedCaches, err := tools.PruneThemeCache()
		if err != nil {
			return err
		}
		fmt.Printf("removed %d expired theme cache(s)\n", removedCaches)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
