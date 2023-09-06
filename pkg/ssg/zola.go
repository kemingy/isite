package ssg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg"
)

const zolaPostTemplate = `
+++
title = "{{ .Title }}"
date = "{{ .CreatedAt }}"
authors = ["{{ .User.Login }}"]
[taxonomies]
tags = [{{ range .Labels }} "{{ .Name }}", {{ end }}]
+++

{{ .Body }}
`

const zolaConfigTemplate = `
title = "{{ .Title }}"
base_url = "{{ .BaseUrl }}"
theme = "{{ .Theme }}"
generate_feed = {{ .Feed }}
taxonomies = [
	{{ range .Taxonomies }} {name = "{{ . }}"}, {{ end }}
]

[markdown]
highlight_code = true
render_emoji = true
`

type Zola struct {
	Title      string
	BaseUrl    string
	Theme      string
	Feed       bool
	Taxonomies []string
}

func (z *Zola) Generate(issues []pkg.Issue, outputDir string) error {
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to get SSG absolute path")
	}
	err = os.MkdirAll(path, os.ModeDir|0755)
	if err != nil {
		return errors.Wrap(err, "failed to create SSG output directory")
	}

	config, err := template.New("config").Parse(zolaConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse SSG config template")
	}
	var configBuf bytes.Buffer
	err = config.Execute(&configBuf, z)
	if err != nil {
		return errors.Wrap(err, "failed to execute SSG config template")
	}

	err = os.WriteFile(filepath.Join(path, "config.toml"), configBuf.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write SSG config file")
	}
	err = os.MkdirAll(filepath.Join(path, "content", "post"), os.ModeDir|0755)
	if err != nil {
		return errors.Wrap(err, "failed to create SSG post directory")
	}

	post, err := template.New("post").Parse(zolaPostTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse SSG post template")
	}
	for _, issue := range issues {
		var postBuf bytes.Buffer
		err = post.Execute(&postBuf, issue)
		if err != nil {
			return errors.Wrap(err, "failed to execute SSG post template")
		}
		err = os.WriteFile(
			filepath.Join(path, "content", "post", fmt.Sprintf("issue-%d.md", issue.Number)),
			postBuf.Bytes(),
			0644)
		if err != nil {
			return errors.Wrap(err, "failed to write SSG post file")
		}
	}
	return nil
}
