package pkg

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/ssg"
	"github.com/kemingy/isite/pkg/types"
)

type issueFilterOption struct {
	State   string
	Creator string
	Labels  []string
}

func (o *issueFilterOption) BuildQuery() string {
	query := fmt.Sprintf("state=%s", o.State)
	if o.Creator == "" {
		query += fmt.Sprintf("&creator=%s", o.Creator)
	}
	if len(o.Labels) > 0 {
		query += fmt.Sprintf("&labels=%s", strings.Join(o.Labels, ","))
	}
	return query
}

type Website struct {
	User string
	Repo string
	// options
	FilterOption issueFilterOption
	PerPage      int
	// data
	Issues []types.Issue
}

type IssueFilterOption interface {
	SetFilterOption(*issueFilterOption)
}

func NewWebsite(user, repo string, opts ...IssueFilterOption) *Website {
	option := issueFilterOption{
		State:   "open",
		Creator: user,
		Labels:  []string{},
	}
	for _, opt := range opts {
		opt.SetFilterOption(&option)
	}
	return &Website{
		User:         user,
		Repo:         repo,
		FilterOption: option,
		PerPage:      100,
	}
}

type IssueFilterByCreator struct {
	Creator string
}

func (f *IssueFilterByCreator) SetFilterOption(option *issueFilterOption) {
	if f.Creator != "" {
		option.Creator = f.Creator
	}
}

type IssueFilterByLabels struct {
	Labels []string
}

func (f *IssueFilterByLabels) SetFilterOption(option *issueFilterOption) {
	for _, label := range f.Labels {
		if label != "" {
			option.Labels = append(option.Labels, label)
		}
	}
}

type IssueFilterByState struct {
	State string
}

func (f *IssueFilterByState) SetFilterOption(option *issueFilterOption) {
	for _, state := range []string{"open", "closed", "all"} {
		if state == f.State {
			option.State = f.State
			return
		}
	}
}

func (w *Website) IssueUrl() string {
	return fmt.Sprintf(
		"repos/%s/%s/issues?%s&per_page=%d", w.User, w.Repo, w.FilterOption.BuildQuery(), w.PerPage)
}

func (w *Website) Retrieve() error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}
	err = client.Get(w.IssueUrl(), &w.Issues)
	if err != nil {
		return err
	}
	return nil
}

func (w *Website) Generate(engine, title, theme, themeRepo, baseUrl, output string, feed bool) error {
	if title == "" {
		title = w.Repo
	}
	generator := ssg.NewGenerator(engine, title, theme, themeRepo, baseUrl, feed)
	err := generator.Generate(w.Issues, "output")
	if err != nil {
		return errors.Wrap(err, "failed to generate static site")
	}
	return nil
}
