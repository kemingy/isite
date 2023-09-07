package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kemingy/isite/pkg"
)

var (
	creator string
	state   string
	label   []string
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
}

func generate(cmd *cobra.Command, args []string) error {
	website := pkg.NewWebsite(
		user, repo,
		&pkg.IssueFilterByCreator{Creator: creator},
		&pkg.IssueFilterByState{State: state},
		&pkg.IssueFilterByLabels{Labels: label},
	)
	err := website.Retrieve()
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", website.Issues)
	return nil
}
