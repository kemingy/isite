package ssg

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
	"github.com/kemingy/isite/pkg/tools"
)

const (
	hugoDefaultTheme     = "PaperMod"
	hugoDefaultThemeRepo = "adityatelange/hugo-PaperMod"
	hugoContentDir       = "content/posts"
	hugoAboutFile        = "content/about.md"
	hugoSearchFile       = "content/search.md"
	hugoOGDir            = "static/images/og"
)

// Hugo and PaperMod both render the Markdown body as-is. Keeping reactions
// and comments in the body makes them work with upstream PaperMod and with
// compatible forks, without requiring a fork just for an isite data model.
const hugoPostTemplate = `+++
title = {{ toml_escape .Title }}
date = {{ toml_escape .CreatedAt }}
lastmod = {{ toml_escape .UpdatedAt }}
author = {{ toml_escape .User.Login }}
tags = [{{ range .Labels }}{{ toml_escape .Name }}, {{ end }}]
canonicalURL = {{ toml_escape .URL }}
showToc = true
tocOpen = false

[cover]
image = "/images/og/issue-{{ .Number }}.svg"
alt = {{ toml_escape .Title }}
hidden = true
+++

{{ if .URL }}> Originally published as [GitHub issue #{{ .Number }}]({{ .URL }}).

{{ end }}{{ .Body }}
{{ with hugo_reactions .Reactions }}
## Reactions

{{ . }}
{{ end }}{{ if .Comments }}
## Comments
{{ range .Comments }}
### {{ .User.Login }}

{{ if .HTMLURL }}[View comment]({{ .HTMLURL }}) · {{ .UpdatedAt }}

{{ end }}{{ .Body }}
{{ end }}{{ end }}
`

const hugoConfigTemplate = `baseURL = {{ toml_escape .BaseURL }}
languageCode = "en-us"
title = {{ toml_escape .Title }}
theme = {{ toml_escape .ThemeName }}
pagination.pagerSize = 10
enableRobotsTXT = true
enableEmoji = true
enableGitInfo = false

[params]
description = {{ toml_escape .Description }}
defaultTheme = "auto"
ShowReadingTime = true
ShowShareButtons = true
ShowPostNavLinks = true
ShowBreadCrumbs = true
ShowCodeCopyButtons = true
comments = false
disableSpecial1stPost = true

[[menu.main]]
name = "Home"
url = "/"
weight = 10

[[menu.main]]
name = "Tags"
url = "/tags/"
weight = 20

[[menu.main]]
name = "Search"
url = "/search/"
weight = 30

[[menu.main]]
name = "About"
url = "/about/"
weight = 40

[taxonomies]
tag = "tags"

[outputs]
home = ["HTML", "JSON"{{ if .Feed }}, "RSS"{{ end }}]
`

type Hugo struct {
	Title       string
	BaseURL     string
	ThemeName   string
	ThemeRepo   string
	Description string
	Feed        bool
	Katex       bool
}

func NewHugo(cmd *models.Command, meta *models.Repository) *Hugo {
	theme, themeRepo := cmd.Theme, cmd.ThemeRepo
	if theme == "" && themeRepo == "" {
		theme, themeRepo = hugoDefaultTheme, hugoDefaultThemeRepo
	}
	description := cmd.Title
	if meta != nil && meta.Description != "" {
		description = meta.Description
	}
	return &Hugo{Title: cmd.Title, BaseURL: cmd.BaseURL, ThemeName: theme, ThemeRepo: themeRepo, Description: description, Feed: cmd.Feed, Katex: cmd.Katex}
}

func (h *Hugo) Generate(issues []models.Issue, outputDir string) error {
	if err := validateOutputDir(outputDir); err != nil {
		return err
	}
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}
	for _, dir := range []string{"themes", hugoContentDir, hugoOGDir} {
		if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
			return errors.Wrapf(err, "failed to create Hugo %s directory", dir)
		}
	}
	if _, err := tools.CloneTheme(h.ThemeRepo, filepath.Join(path, "themes", h.ThemeName), "theme.toml"); err != nil {
		return err
	}
	if err := h.removeLegacyZolaPosts(path); err != nil {
		return err
	}
	if err := h.writeConfig(path); err != nil {
		return err
	}
	if err := h.writePages(path); err != nil {
		return err
	}
	return h.writePosts(path, issues)
}

func (h *Hugo) removeLegacyZolaPosts(path string) error {
	matches, err := filepath.Glob(filepath.Join(path, "content", "issue-*.md"))
	if err != nil {
		return errors.Wrap(err, "failed to find legacy Zola posts")
	}
	for _, name := range matches {
		if err := os.Remove(name); err != nil {
			return errors.Wrapf(err, "failed to remove legacy Zola post %s", name)
		}
	}
	return nil
}

func (h *Hugo) writeConfig(path string) error {
	t, err := template.New("hugo-config").Funcs(template.FuncMap{"toml_escape": tools.EscapeTOMLString}).Parse(hugoConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse Hugo config template")
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, h); err != nil {
		return errors.Wrap(err, "failed to execute Hugo config template")
	}
	if err := os.WriteFile(filepath.Join(path, "hugo.toml"), buf.Bytes(), 0644); err != nil {
		return errors.Wrap(err, "failed to write Hugo config file")
	}
	return nil
}

func (h *Hugo) writePosts(path string, issues []models.Issue) error {
	t, err := template.New("hugo-post").Funcs(template.FuncMap{"toml_escape": tools.EscapeTOMLString, "hugo_reactions": hugoReactionText}).Parse(hugoPostTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse Hugo post template")
	}
	dir := filepath.Join(path, hugoContentDir)
	for _, issue := range issues {
		var buf bytes.Buffer
		if err := t.Execute(&buf, issue); err != nil {
			return errors.Wrapf(err, "failed to execute Hugo post template for issue #%d", issue.Number)
		}
		name := filepath.Join(dir, fmt.Sprintf("issue-%d.md", issue.Number))
		if err := os.WriteFile(name, buf.Bytes(), 0644); err != nil {
			return errors.Wrapf(err, "failed to write Hugo post for issue #%d", issue.Number)
		}
		if err := h.writeOGImage(path, issue); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hugo) writeOGImage(path string, issue models.Issue) error {
	title := html.EscapeString(issue.Title)
	siteTitle := html.EscapeString(h.Title)
	titleLines := ogTitleLines(issue.Title, 30)
	var titleSVG strings.Builder
	for index, line := range titleLines {
		line = html.EscapeString(line)
		if index == 0 {
			fmt.Fprintf(&titleSVG, `<tspan x="108" y="245">%s</tspan>`, line)
		} else {
			fmt.Fprintf(&titleSVG, `<tspan x="108" dy="72">%s</tspan>`, line)
		}
	}
	fontFamily := `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", "Noto Sans CJK TC", "PingFang SC", "Microsoft YaHei", Arial, sans-serif`
	content := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630" role="img" aria-labelledby="title subtitle">
<title id="title">%s</title>
<desc id="subtitle">%s</desc>
<rect width="1200" height="630" fill="#1d1f21"/>
<rect x="54" y="54" width="12" height="522" rx="6" fill="#ffcc66"/>
<text x="108" y="145" fill="#ffcc66" font-family="%s" font-size="30" font-weight="600">%s</text>
<text fill="#ffffff" font-family="%s" font-size="62" font-weight="700">%s</text>
<text x="108" y="540" fill="#b8b8b8" font-family="%s" font-size="26">GitHub issue #%d</text>
</svg>
`, title, title, fontFamily, siteTitle, fontFamily, titleSVG.String(), fontFamily, issue.Number)
	name := filepath.Join(path, hugoOGDir, fmt.Sprintf("issue-%d.svg", issue.Number))
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		return errors.Wrapf(err, "failed to write Hugo OG image for issue #%d", issue.Number)
	}
	return nil
}

func ogTitleLines(title string, maxRunes int) []string {
	if maxRunes < 1 {
		return []string{title}
	}
	words := strings.Fields(title)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, (len([]rune(title))/maxRunes)+1)
	line := ""
	for _, word := range words {
		if len([]rune(word)) > maxRunes {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			runes := []rune(word)
			for len(runes) > maxRunes {
				lines = append(lines, string(runes[:maxRunes]))
				runes = runes[maxRunes:]
			}
			line = string(runes)
			continue
		}
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if len([]rune(candidate)) > maxRunes && line != "" {
			lines = append(lines, line)
			line = word
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func (h *Hugo) writePages(path string) error {
	pages := map[string]string{
		filepath.Join(path, hugoAboutFile): `+++
title = "About"
description = ` + tools.EscapeTOMLString(h.Description) + `
+++

` + h.Description + `
`,
		filepath.Join(path, hugoSearchFile): `+++
title = "Search"
layout = "search"
summary = "Search posts"
+++
`,
	}
	for name, content := range pages {
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			return errors.Wrapf(err, "failed to write Hugo page %s", name)
		}
	}
	return nil
}

func hugoReactionText(reactions models.Reactions) string {
	items := []struct {
		emoji string
		count int
	}{
		{"👍", reactions.ThumbUp}, {"👎", reactions.ThumbDown}, {"😄", reactions.Laugh},
		{"🎉", reactions.Hooray}, {"😕", reactions.Confused}, {"❤️", reactions.Heart},
		{"🚀", reactions.Rocket}, {"👀", reactions.Eyes},
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", item.emoji, item.count))
		}
	}
	return strings.Join(parts, " · ")
}
