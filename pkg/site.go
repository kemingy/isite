package pkg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
	"github.com/kemingy/isite/pkg/ssg"
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
	Issues []models.Issue
	// others
	linkRegex *regexp.Regexp
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
	linkRegex := regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
	return &Website{
		User:         user,
		Repo:         repo,
		FilterOption: option,
		PerPage:      100,
		Issues:       []models.Issue{},
		linkRegex:    linkRegex,
	}
}

type IssueFilterByCreator struct {
	Creator string
}

func (f *IssueFilterByCreator) SetFilterOption(option *issueFilterOption) {
	if f.Creator != "" {
		option.Creator = url.QueryEscape(f.Creator)
	}
}

type IssueFilterByLabels struct {
	Labels []string
}

func (f *IssueFilterByLabels) SetFilterOption(option *issueFilterOption) {
	for _, label := range f.Labels {
		if label != "" {
			option.Labels = append(option.Labels, url.QueryEscape(label))
		}
	}
}

type IssueFilterByState struct {
	State string
}

func (f *IssueFilterByState) SetFilterOption(option *issueFilterOption) {
	for _, state := range []string{"open", "closed", "all"} {
		if state == f.State {
			option.State = url.QueryEscape(f.State)
			return
		}
	}
}

func (w *Website) IssueURL() string {
	return fmt.Sprintf(
		"repos/%s/%s/issues?%s&per_page=%d", w.User, w.Repo, w.FilterOption.BuildQuery(), w.PerPage)
}

func (w *Website) CommentURL(issueNumber int) string {
	return fmt.Sprintf("repos/%s/%s/issues/%d/comments?per_page=%d", w.User, w.Repo, issueNumber, w.PerPage)
}

func (w *Website) findNextPage(response *http.Response) (string, bool) {
	for _, m := range w.linkRegex.FindAllStringSubmatch(response.Header.Get("Link"), -1) {
		if len(m) > 2 && m[2] == "next" {
			return m[1], true
		}
	}
	return "", false
}

func (w *Website) Retrieve() error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}

	// with pagination: https://github.com/cli/go-gh/blob/d32c104a9a25c9de3d7c7b07a43ae0091441c858/example_gh_test.go#L96
	url := w.IssueURL()
	var hasNextPage bool
	for {
		response, err := client.Request(http.MethodGet, url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to get issues")
		}
		issues := []models.Issue{}
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&issues)
		if err != nil {
			return errors.Wrap(err, "failed to decode issues")
		}
		// GitHub's REST API considers every pull request an issue, but not every issue is a pull request.
		for _, issue := range issues {
			// Identify pull requests by the `pull_request` key.
			if issue.PullRequest != nil {
				continue
			}
			w.Issues = append(w.Issues, issue)
		}
		if err := response.Body.Close(); err != nil {
			return err
		}
		if url, hasNextPage = w.findNextPage(response); !hasNextPage {
			break
		}
	}

	// comments
	for i, issue := range w.Issues {
		url = w.CommentURL(issue.Number)
		w.Issues[i].Comments = []models.Comment{}
		for {
			response, err := client.Request(http.MethodGet, url, nil)
			if err != nil {
				return errors.Wrapf(err, "failed to get comments for issue #%d", issue.Number)
			}
			comments := []models.Comment{}
			decoder := json.NewDecoder(response.Body)
			err = decoder.Decode(&comments)
			if err != nil {
				return errors.Wrapf(err, "failed to decode comments for issue #%d", issue.Number)
			}
			w.Issues[i].Comments = append(w.Issues[i].Comments, comments...)
			if err := response.Body.Close(); err != nil {
				return err
			}
			if url, hasNextPage = w.findNextPage(response); !hasNextPage {
				break
			}
		}
	}
	return nil
}

func (w *Website) Generate(engine, title, theme, themeRepo, baseURL, output string, feed bool) error {
	if title == "" {
		title = w.Repo
	}
	generator := ssg.NewGenerator(engine, title, theme, themeRepo, baseURL, feed)
	err := generator.Generate(w.Issues, output)
	if err != nil {
		return errors.Wrap(err, "failed to generate static site")
	}
	return nil
}
