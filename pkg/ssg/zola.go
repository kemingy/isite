package ssg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/types"
	"github.com/kemingy/isite/pkg/utils"
)

const (
	zolaDefaultTheme     = "hyde"
	zolaDefaultThemeRepo = "getzola/hyde"
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
theme = "{{ .ThemeName }}"
compile_sass = true
generate_feed = {{ .Feed }}
taxonomies = [
	{{ range .Taxonomies }}{ name = "{{ . }}"},{{ end }}
]

[markdown]
highlight_code = true
render_emoji = true

[extra]
`

type Zola struct {
	Title      string
	BaseUrl    string
	ThemeName  string
	ThemeRepo  string
	Feed       bool
	Taxonomies []string
}

func NewZola(title, baseUrl, theme, themeRepo string, feed bool) *Zola {
	if theme == "" && themeRepo == "" {
		theme = zolaDefaultTheme
		themeRepo = zolaDefaultThemeRepo
	}

	return &Zola{
		Title:      title,
		BaseUrl:    baseUrl,
		ThemeName:  theme,
		ThemeRepo:  themeRepo,
		Feed:       feed,
		Taxonomies: []string{"tags"},
	}
}

func (z *Zola) generateDir(path string) error {
	err := os.MkdirAll(path, os.ModeDir|0755)
	if err != nil {
		return errors.Wrap(err, "failed to create zola output directory")
	}

	for _, dir := range []string{"themes", "templates", "content"} {
		err = os.MkdirAll(filepath.Join(path, dir), os.ModeDir|0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create zola %s directory", dir)
		}
	}
	return nil
}

func (z *Zola) downloadTheme(path string) error {
	return utils.CloneTheme(z.ThemeRepo, filepath.Join(path, "themes", z.ThemeName))
}

func (z *Zola) generateConfig(path string) error {
	config, err := template.New("config").Parse(zolaConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse zola config template")
	}
	var configBuf bytes.Buffer
	err = config.Execute(&configBuf, z)
	if err != nil {
		return errors.Wrap(err, "failed to execute zola config template")
	}

	err = os.WriteFile(filepath.Join(path, "config.toml"), configBuf.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write zola config file")
	}
	return nil
}

func (z *Zola) generatePost(path string, issues []types.Issue) error {
	post, err := template.New("post").Parse(zolaPostTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse zola post template")
	}
	for _, issue := range issues {
		var postBuf bytes.Buffer
		err = post.Execute(&postBuf, issue)
		if err != nil {
			return errors.Wrap(err, "failed to execute zola post template")
		}
		err = os.WriteFile(
			filepath.Join(path, "content", fmt.Sprintf("issue-%d.md", issue.Number)),
			postBuf.Bytes(),
			0644)
		if err != nil {
			return errors.Wrap(err, "failed to write zola post file")
		}
	}
	return nil
}

func (z *Zola) Generate(issues []types.Issue, outputDir string) error {
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}

	if err = z.generateDir(path); err != nil {
		return err
	}
	if err = z.downloadTheme(path); err != nil {
		return err
	}
	if err = z.generateConfig(path); err != nil {
		return err
	}
	if err = z.generatePost(path, issues); err != nil {
		return err
	}
	return nil
}
