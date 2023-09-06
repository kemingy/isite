package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/kemingy/isite/pkg"
)

var (
	user    string
	repo    string
	creator string
	state   string
	label   string
)

func init() {
	flag.StringVar(&user, "user", "kemingy", "github user name or organization name")
	flag.StringVar(&repo, "repo", "isite", "github repository name")
	flag.StringVar(&creator, "creator", "", "filter the github issue by the creator")
	flag.StringVar(&state, "state", "open", "filter the github issue by the state, default is `open`, choose from [open, closed, all]")
	flag.StringVar(&label, "label", "", "filter the github issue by the label")
}

func main() {
	flag.Parse()
	website := pkg.NewWebsite(
		user, repo,
		&pkg.IssueFilterByCreator{Creator: creator},
		&pkg.IssueFilterByState{State: state},
		&pkg.IssueFilterByLabels{Labels: []string{label}},
	)
	fmt.Println(website.IssueUrl())
	err := website.Retrieve()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", website.Issues)
}
