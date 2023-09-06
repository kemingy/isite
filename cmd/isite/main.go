package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/kemingy/isite/pkg"
)

var (
	user string
	repo string
)

func init() {
	flag.StringVar(&user, "user", "kemingy", "github user name or organization name")
	flag.StringVar(&repo, "repo", "isite", "github repository name")
	flag.Parse()
}

func main() {
	client, err := api.DefaultRESTClient()
	if err != nil {
		log.Fatal(err)
	}
	var response []pkg.Issue
	err = client.Get(fmt.Sprintf("repos/%s/%s/issues", user, repo), &response)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", response)
}
