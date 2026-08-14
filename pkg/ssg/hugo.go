package ssg

import (
	"bytes"
	"encoding/json"
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
	hugoDefaultTheme      = "PaperMod"
	hugoDefaultThemeRepo  = "adityatelange/hugo-PaperMod"
	hugoContentDir        = "content/posts"
	hugoAboutFile         = "content/about.md"
	hugoSearchFile        = "content/search.md"
	hugoOGDir             = "static/images/og"
	hugoEngagementPartial = "layouts/_partials/extend_post_content.html"
	hugoStylesheet        = "assets/css/extended/isite.css"
	hugoCommentsDataDir   = "data/comments"
)

// Hugo and PaperMod render the Markdown body as-is. Engagement data is kept
// in front matter so the generated partial can render it outside the ToC.
const hugoPostTemplate = `+++
title = {{ toml_escape .Title }}
date = {{ toml_escape .CreatedAt }}
lastmod = {{ toml_escape .UpdatedAt }}
author = {{ toml_escape .User.Login }}
tags = {{ hugo_tags .Labels }}
canonicalURL = {{ toml_escape .URL }}
issueURL = {{ toml_escape .URL }}
showToc = true
tocOpen = false

[params]
commentCount = {{ len .Comments }}
commentsKey = "issue-{{ .Number }}"
[params.reactions]
thumbs_up = {{ .Reactions.ThumbUp }}
thumbs_down = {{ .Reactions.ThumbDown }}
laugh = {{ .Reactions.Laugh }}
hooray = {{ .Reactions.Hooray }}
confused = {{ .Reactions.Confused }}
heart = {{ .Reactions.Heart }}
rocket = {{ .Reactions.Rocket }}
eyes = {{ .Reactions.Eyes }}

[cover]
image = "/images/og/issue-{{ .Number }}.svg"
alt = {{ toml_escape .Title }}
hidden = true
+++

{{ if .URL }}> Originally published as [GitHub issue #{{ .Number }}]({{ .URL }}).

{{ end }}{{ .Body }}
`

const hugoConfigTemplate = `baseURL = {{ toml_escape .BaseURL }}
locale = "en-us"
title = {{ toml_escape .Title }}
theme = {{ toml_escape .ThemeName }}
pagination.pagerSize = 10
enableRobotsTXT = true
enableEmoji = true
enableGitInfo = false

[params]
env = "production"
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

[markup.goldmark.renderer]
unsafe = true

[outputs]
home = ["HTML", "JSON"{{ if .Feed }}, "RSS"{{ end }}]
`

const hugoAboutPageTemplate = `+++
title = "About"
description = %s
+++

%s
`

const hugoSearchPageTemplate = `+++
title = "Search"
layout = "search"
summary = "Search posts"
+++
`

const hugoEngagementTemplate = `{{- with .Params.reactions }}
{{- if or (gt .thumbs_up 0) (gt .thumbs_down 0) (gt .laugh 0) (gt .hooray 0) (gt .confused 0) (gt .heart 0) (gt .rocket 0) (gt .eyes 0) }}
<div class="isite-reactions" aria-label="Reactions">
  {{- if gt .thumbs_up 0 }}<span>👍 {{ .thumbs_up }}</span>{{ end }}
  {{- if gt .thumbs_down 0 }}<span>👎 {{ .thumbs_down }}</span>{{ end }}
  {{- if gt .laugh 0 }}<span>😄 {{ .laugh }}</span>{{ end }}
  {{- if gt .hooray 0 }}<span>🎉 {{ .hooray }}</span>{{ end }}
  {{- if gt .confused 0 }}<span>😕 {{ .confused }}</span>{{ end }}
  {{- if gt .heart 0 }}<span>❤️ {{ .heart }}</span>{{ end }}
  {{- if gt .rocket 0 }}<span>🚀 {{ .rocket }}</span>{{ end }}
  {{- if gt .eyes 0 }}<span>👀 {{ .eyes }}</span>{{ end }}
</div>
{{- end }}
{{- end }}
{{- with .Params.commentsKey }}
{{- with (index site.Data.comments .) }}
<section class="isite-comments" aria-labelledby="comments-heading">
  <h2 id="comments-heading">Comments <span>{{ len . }}</span></h2>
  <p class="isite-comments-note">Read-only mirror of the comments on
    {{ if $.Params.issueURL }} <a href="{{ $.Params.issueURL }}">this GitHub issue</a>{{ else }} the GitHub issue{{ end }}.
  </p>
  {{- range . }}
  <article class="isite-comment">
    <header class="isite-comment-header">
      {{- if .URL }}<a href="{{ .URL }}">{{ end }}
      {{- if .Avatar }}<img src="{{ .Avatar }}" alt="" width="40" height="40">{{ end }}
      <div><strong>{{ .Author }}</strong><span>{{ .Updated }}</span></div>
      {{- if .URL }}</a>{{ end }}
    </header>
    <div class="isite-comment-body md-content">{{ .Body | markdownify }}</div>
  </article>
  {{- end }}
</section>
{{- end }}
{{- end }}
`

const hugoStylesheetTemplate = `.isite-reactions {
  display: flex;
  flex-wrap: wrap;
  gap: .5rem;
  margin: 1.5rem 0;
}
.isite-reactions span {
  border: 1px solid var(--tertiary);
  border-radius: 999px;
  padding: .25rem .65rem;
  color: var(--secondary);
}
.isite-comments {
  border-top: 1px solid var(--tertiary);
  margin-top: 2rem;
  padding-top: 1.5rem;
}
.isite-comments h2 { font-size: 1.35rem; }
.isite-comments h2 span { color: var(--secondary); font-size: .9em; }
.isite-comments-note { color: var(--secondary); font-size: .9em; }
.isite-comment {
  border: 1px solid var(--tertiary);
  border-radius: .5rem;
  margin: 1rem 0;
  overflow: hidden;
}
.isite-comment-header {
  align-items: center;
  background: var(--code-bg);
  display: flex;
  gap: .75rem;
  padding: .75rem 1rem;
}
.isite-comment-header a { align-items: center; color: inherit; display: flex; gap: .75rem; width: 100%; }
.isite-comment-header img { border-radius: 50%; }
.isite-comment-header div { display: flex; flex-direction: column; }
.isite-comment-header span { color: var(--secondary); font-size: .85em; }
.isite-comment-body { padding: 1rem; }
.isite-comment-body > :first-child { margin-top: 0; }
.isite-comment-body > :last-child { margin-bottom: 0; }
`

type Hugo struct {
	Title         string
	BaseURL       string
	ThemeName     string
	ThemeRepo     string
	ThemeRevision string
	Description   string
	Feed          bool
	Katex         bool
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
	themeRevision := ""
	if cmd.ThemeRevision != nil {
		themeRevision = *cmd.ThemeRevision
	}
	return &Hugo{
		Title:         cmd.Title,
		BaseURL:       cmd.BaseURL,
		ThemeName:     theme,
		ThemeRepo:     themeRepo,
		ThemeRevision: themeRevision,
		Description:   description,
		Feed:          cmd.Feed,
		Katex:         cmd.Katex,
	}
}

func (h *Hugo) Generate(issues []models.Issue, outputDir string) error {
	if err := validateOutputDir(outputDir); err != nil {
		return err
	}
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}
	for _, dir := range []string{"themes", hugoContentDir, hugoOGDir, filepath.Dir(hugoEngagementPartial), filepath.Dir(hugoStylesheet), hugoCommentsDataDir} {
		if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
			return errors.Wrapf(err, "failed to create Hugo %s directory", dir)
		}
	}
	if _, err := tools.CloneTheme(h.ThemeRepo, filepath.Join(path, "themes", h.ThemeName), h.ThemeRevision, "theme.toml"); err != nil {
		return err
	}
	if err := h.writeConfig(path); err != nil {
		return err
	}
	if err := h.writePages(path); err != nil {
		return err
	}
	if err := h.writeEngagementPartial(path); err != nil {
		return err
	}
	if err := h.writeStylesheet(path); err != nil {
		return err
	}
	return h.writePosts(path, issues)
}

func (h *Hugo) writeConfig(path string) error {
	t, err := template.New("hugo-config").Funcs(template.FuncMap{templateTOMLEscape: tools.EscapeTOMLString}).Parse(hugoConfigTemplate)
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
	t, err := template.New("hugo-post").Funcs(template.FuncMap{
		templateTOMLEscape: tools.EscapeTOMLString,
		"hugo_tags":        hugoTags,
	}).Parse(hugoPostTemplate)
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
		if err := h.writeCommentsData(path, issue); err != nil {
			return err
		}
	}
	return nil
}

func hugoTags(labels []models.Label) string {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		values = append(values, tools.EscapeTOMLString(label.Name))
	}
	return "[" + strings.Join(values, ", ") + "]"
}

func (h *Hugo) writeOGImage(path string, issue models.Issue) error {
	title := html.EscapeString(issue.Title)
	siteTitle := html.EscapeString(h.Title)
	author := html.EscapeString(issue.User.Login)
	titleLines := ogTitleLines(issue.Title, 30)
	if len(titleLines) > 4 {
		titleLines = titleLines[:4]
		last := []rune(titleLines[3])
		if len(last) > 0 {
			titleLines[3] = string(last[:len(last)-1]) + "…"
		}
	}
	var titleSVG strings.Builder
	for index, line := range titleLines {
		line = html.EscapeString(line)
		if index == 0 {
			fmt.Fprintf(&titleSVG, `<tspan x="108" y="290">%s</tspan>`, line)
		} else {
			fmt.Fprintf(&titleSVG, `<tspan x="108" dy="58">%s</tspan>`, line)
		}
	}
	// Use portable system fallbacks. SVGs do not embed fonts, so the viewer
	// selects the first installed font that supports the title's characters.
	fontFamily := `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "DejaVu Sans", "Liberation Sans", "FreeSans", "Noto Sans CJK SC", "Noto Sans CJK TC", "Noto Sans CJK JP", "Noto Sans CJK KR", "PingFang SC", "Microsoft YaHei", "WenQuanYi Zen Hei", Arial, sans-serif`
	content := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630" role="img" aria-labelledby="title subtitle">
<title id="title">%s</title>
<desc id="subtitle">%s</desc>
<rect width="1200" height="630" fill="#f6f8fa"/>
<rect width="1200" height="166" fill="#24292f"/>
<rect y="160" width="1200" height="6" fill="#2da44e"/>
<text x="72" y="103" fill="#ffffff" font-family='%s' font-size="36" font-weight="700">%s</text>
<text x="1128" y="101" fill="#8b949e" font-family='%s' font-size="24" font-weight="600" text-anchor="end">ISSUE #%d</text>
<rect x="72" y="218" width="1056" height="300" rx="14" fill="#ffffff" stroke="#d0d7de" stroke-width="2"/>
<text fill="#1f2328" font-family='%s' font-size="54" font-weight="700">%s</text>
<line x1="108" y1="480" x2="1092" y2="480" stroke="#d8dee4" stroke-width="2"/>
<text x="108" y="518" fill="#57606a" font-family='%s' font-size="24">GitHub issue #%d</text>
<text x="1092" y="518" fill="#57606a" font-family='%s' font-size="24" text-anchor="end">%s</text>
</svg>
`, title, title, fontFamily, siteTitle, fontFamily, issue.Number, fontFamily, titleSVG.String(), fontFamily, issue.Number, fontFamily, author)
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
		filepath.Join(path, hugoAboutFile):  fmt.Sprintf(hugoAboutPageTemplate, tools.EscapeTOMLString(h.Description), h.Description),
		filepath.Join(path, hugoSearchFile): hugoSearchPageTemplate,
	}
	for name, content := range pages {
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			return errors.Wrapf(err, "failed to write Hugo page %s", name)
		}
	}
	return nil
}

func (h *Hugo) writeStylesheet(path string) error {
	if err := os.WriteFile(filepath.Join(path, hugoStylesheet), []byte(hugoStylesheetTemplate), 0644); err != nil {
		return errors.Wrap(err, "failed to write Hugo stylesheet")
	}
	return nil
}

type hugoComment struct {
	Author  string `json:"author"`
	Avatar  string `json:"avatar"`
	URL     string `json:"url"`
	Updated string `json:"updated"`
	Body    string `json:"body"`
}

func (h *Hugo) writeCommentsData(path string, issue models.Issue) error {
	comments := make([]hugoComment, 0, len(issue.Comments))
	for _, comment := range issue.Comments {
		comments = append(comments, hugoComment{
			Author: comment.User.Login, Avatar: comment.User.AvatarURL,
			URL: comment.HTMLURL, Updated: comment.UpdatedAt, Body: comment.Body,
		})
	}
	content, err := json.Marshal(comments)
	if err != nil {
		return errors.Wrapf(err, "failed to encode Hugo comments for issue #%d", issue.Number)
	}
	name := filepath.Join(path, hugoCommentsDataDir, fmt.Sprintf("issue-%d.json", issue.Number))
	if err := os.WriteFile(name, content, 0644); err != nil {
		return errors.Wrapf(err, "failed to write Hugo comments for issue #%d", issue.Number)
	}
	return nil
}

func (h *Hugo) writeEngagementPartial(path string) error {
	name := filepath.Join(path, hugoEngagementPartial)
	if err := os.WriteFile(name, []byte(hugoEngagementTemplate), 0644); err != nil {
		return errors.Wrap(err, "failed to write Hugo engagement partial")
	}
	return nil
}
