package cli

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	user string
	repo string
)

var rootCmd = &cobra.Command{
	Use:   "isite",
	Short: "isite is a tool to generate static site from github issue",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&user, "user", "kemingy", "github user name or organization name")
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "isite", "github repository name")
}
