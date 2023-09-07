package cli

import (
	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg/version"
)

var (
	short bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of isite",
	Run:   commandVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "print the version number only")
}

func commandVersion(cmd *cobra.Command, _ []string) {
	ver := version.GetVersionInfo()
	if short {
		cmd.Println(ver.Version)
		return
	}
	cmd.Println(ver.PrettyString())
}
