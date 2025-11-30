package cli

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg"
	"github.com/kemingy/isite/pkg/models"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate static site from github issue",
	RunE:  generate,
}
var cmd = &models.Command{}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&cmd.Creator, "creator", "", "filter the github issue by the creator")
	generateCmd.Flags().StringVar(&cmd.State, "state", "open", "filter the github issue by the state, default is `open`, choose from [open, closed, all]")
	generateCmd.Flags().StringSliceVar(&cmd.Label, "label", []string{}, "filter the github issue by the labels")

	generateCmd.Flags().StringVar(&cmd.Engine, "engine", "zola", "the static site generator engine, default is `zola`, choose from [zola]")
	generateCmd.Flags().StringVar(&cmd.Output, "output", "output", "the output dir for the generated files")
	generateCmd.Flags().StringVar(&cmd.Title, "title", "", "the title of the static site, if not set, will use the repository name")
	generateCmd.Flags().StringVar(&cmd.Theme, "theme", "", "the theme name of the static site")
	generateCmd.Flags().StringVar(&cmd.ThemeRepo, "theme-repo", "", "the theme repository of the static site, format is `<user>/<repo>`")
	generateCmd.Flags().StringVar(&cmd.BaseURL, "base-url", "/", "the base url of the static site")
	generateCmd.Flags().BoolVar(&cmd.Feed, "feed", true, "generate feed or not")
	generateCmd.Flags().BoolVar(&cmd.Katex, "katex", false, "enable katex support or not")
}

func generate(_ *cobra.Command, _ []string) error {
	if (cmd.Theme == "" && cmd.ThemeRepo != "") || (cmd.Theme != "" && cmd.ThemeRepo == "") {
		return errors.New("`theme` and `theme-repo` should be set together")
	}

	website := pkg.NewWebsite(
		user, repo,
		&pkg.IssueFilterByCreator{Creator: cmd.Creator},
		&pkg.IssueFilterByState{State: cmd.State},
		&pkg.IssueFilterByLabels{Labels: cmd.Label},
	)
	if err := website.Retrieve(); err != nil {
		return err
	}
	fmt.Printf("found %d issues\n", len(website.Issues))
	return website.Generate(cmd)
}
