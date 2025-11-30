package ssg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
	"github.com/kemingy/isite/pkg/tools"
)

const (
	zolaDefaultTheme     = "even"
	zolaDefaultThemeRepo = "kemingy/even"
)

const zolaPostTemplate = `
+++
title = "{{ .Title }}"
date = "{{ .CreatedAt }}"
authors = ["{{ .User.Login }}"]
[taxonomies]
tags = [{{ range .Labels }} "{{ .Name }}", {{ end }}]
[extra]
author = "{{ .User.Login }}"
avatar = "{{ .User.AvatarURL }}"
issue_url = "{{ .URL }}"
[extra.reactions]
thumbs_up = {{ .Reactions.ThumbUp }}
thumbs_down = {{ .Reactions.ThumbDown }}
laugh = {{ .Reactions.Laugh }}
heart = {{ .Reactions.Heart }}
hooray = {{ .Reactions.Hooray }}
confused = {{ .Reactions.Confused }}
rocket = {{ .Reactions.Rocket }}
eyes = {{ .Reactions.Eyes }}
{{ range .Comments }}
[[extra.comments]]
url = "{{ .HTMLURL }}"
author_name = "{{ .User.Login }}"
author_avatar = "{{ .User.AvatarURL }}"
content = {{ toml_escape .Body }}
updated_at = "{{ .UpdatedAt }}"
{{ end }}
+++

{{ .Body }}
`

const zolaIndexTemplate = `
+++
paginate_by = 10
sort_by = "date"
+++
`

const zolaConfigTemplate = `
title = "{{ .Title }}"
description = "{{ toml_escape .Description }}"
base_url = "{{ .BaseURL }}"
theme = "{{ .ThemeName }}"
compile_sass = true
generate_feeds = {{ .Feed }}
taxonomies = [
	{{ range .Taxonomies }}{ name = "{{ . }}"},{{ end }}
]

[markdown]
highlight_code = true
render_emoji = true

[extra]
# this only affects the default "even" theme
even_menu = [
    {url = "$BASE_URL", name = "Home"},
    {url = "$BASE_URL/tags", name = "Tags"},
]
katex_enable = {{ .Katex }}
`

type Zola struct {
	Title       string
	BaseURL     string
	ThemeName   string
	ThemeRepo   string
	Description string
	Feed        bool
	Katex       bool
	Taxonomies  []string
}

func NewZola(cmd *models.Command, meta *models.Repository) *Zola {
	theme := cmd.Theme
	themeRepo := cmd.ThemeRepo
	if theme == "" && themeRepo == "" {
		theme = zolaDefaultTheme
		themeRepo = zolaDefaultThemeRepo
	}
	description := cmd.Title
	if meta != nil && len(meta.Description) > 0 {
		description = meta.Description
	}

	return &Zola{
		Title:       cmd.Title,
		BaseURL:     cmd.BaseURL,
		ThemeName:   theme,
		ThemeRepo:   themeRepo,
		Description: description,
		Feed:        cmd.Feed,
		Katex:       cmd.Katex,
		Taxonomies:  []string{"tags"},
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
	return tools.CloneTheme(z.ThemeRepo, filepath.Join(path, "themes", z.ThemeName))
}

func (z *Zola) generateConfig(path string) error {
	funcMap := template.FuncMap{
		"toml_escape": tools.EscapeTOMLString,
	}
	config, err := template.New("config").Funcs(funcMap).Parse(zolaConfigTemplate)
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

func (z *Zola) generateIndex(path string) error {
	index, err := template.New("index").Parse(zolaIndexTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse zola index template")
	}
	var indexBuf bytes.Buffer
	err = index.Execute(&indexBuf, z)
	if err != nil {
		return errors.Wrap(err, "failed to execute zola index template")
	}

	err = os.WriteFile(filepath.Join(path, "content", "_index.md"), indexBuf.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write zola index file")
	}
	return nil
}

func (z *Zola) generatePost(path string, issues []models.Issue) error {
	funcMap := template.FuncMap{
		"toml_escape": tools.EscapeTOMLString,
	}
	post, err := template.New("post").Funcs(funcMap).Parse(zolaPostTemplate)
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

func (z *Zola) Generate(issues []models.Issue, outputDir string) error {
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}

	for _, fn := range []func(path string) error{
		z.generateDir, z.downloadTheme, z.generateConfig, z.generateIndex,
	} {
		if err = fn(path); err != nil {
			return err
		}
	}

	return z.generatePost(path, issues)
}
