package cli

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg"
)

var (
	// filter
	creator string
	state   string
	label   []string
	// config
	engine    string
	output    string
	title     string
	theme     string
	themeRepo string
	baseUrl   string
	feed      bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate static site from github issue",
	RunE:  generate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&creator, "creator", "", "filter the github issue by the creator")
	generateCmd.Flags().StringVar(&state, "state", "open", "filter the github issue by the state, default is `open`, choose from [open, closed, all]")
	generateCmd.Flags().StringSliceVar(&label, "label", []string{}, "filter the github issue by the labels")

	generateCmd.Flags().StringVar(&engine, "engine", "zola", "the static site generator engine, default is `zola`, choose from [zola]")
	generateCmd.Flags().StringVar(&output, "output", "output", "the output dir for the generated files")
	generateCmd.Flags().StringVar(&title, "title", "", "the title of the static site, if not set, will use the repository name")
	generateCmd.Flags().StringVar(&theme, "theme", "", "the theme name of the static site")
	generateCmd.Flags().StringVar(&themeRepo, "theme-repo", "", "the theme repository of the static site, format is `<user>/<repo>`")
	generateCmd.Flags().StringVar(&baseUrl, "base-url", "/", "the base url of the static site")
	generateCmd.Flags().BoolVar(&feed, "feed", true, "generate feed or not")
}

func generate(cmd *cobra.Command, args []string) error {
	if (theme == "" && themeRepo != "") || (theme != "" && themeRepo == "") {
		return errors.New("`theme` and `theme-repo` should be set together")
	}

	website := pkg.NewWebsite(
		user, repo,
		&pkg.IssueFilterByCreator{Creator: creator},
		&pkg.IssueFilterByState{State: state},
		&pkg.IssueFilterByLabels{Labels: label},
	)
	if err := website.Retrieve(); err != nil {
		return err
	}
	fmt.Printf("found %d issues\n", len(website.Issues))
	return website.Generate(engine, title, theme, themeRepo, baseUrl, output, feed)
}
