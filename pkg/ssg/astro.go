package ssg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
	"github.com/kemingy/isite/pkg/tools"
)

const (
	astroDefaultTheme     = "astro-paper"
	astroDefaultThemeRepo = "satnaing/astro-paper"
	astroPostsDir         = "src/content/posts"
	astroConfigFile       = "astro.config.ts"
	astroThemeConfigFile  = "astro-paper.config.ts"
)

type Astro struct {
	Title       string
	BaseURL     string
	Description string
	Author      string
	Repository  string
	Feed        bool
	Katex       bool
	Theme       string
	ThemeRepo   string
}

func NewAstro(cmd *models.Command, meta *models.Repository) *Astro {
	theme := cmd.Theme
	themeRepo := cmd.ThemeRepo
	if theme == "" && themeRepo == "" {
		theme = astroDefaultTheme
		themeRepo = astroDefaultThemeRepo
	}
	description := cmd.Title
	author := "GitHub"
	repository := ""
	if meta != nil {
		if meta.Description != "" {
			description = meta.Description
		}
		if meta.Owner.Login != "" {
			author = meta.Owner.Login
		}
		if meta.FullName != "" {
			repository = "https://github.com/" + meta.FullName
		}
	}
	return &Astro{
		Title: cmd.Title, BaseURL: cmd.BaseURL, Description: description,
		Author: author, Repository: repository, Feed: cmd.Feed, Katex: cmd.Katex,
		Theme: theme, ThemeRepo: themeRepo,
	}
}

func (a *Astro) Generate(issues []models.Issue, outputDir string) error {
	if err := validateOutputDir(outputDir); err != nil {
		return err
	}
	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}
	fresh, err := tools.CloneTheme(a.ThemeRepo, path, "package.json")
	if err != nil {
		return err
	}
	if err := a.configureAstroPaper(path, fresh); err != nil {
		return err
	}
	return a.generatePosts(path, issues, fresh)
}

// configureAstroPaper changes only AstroPaper's public configuration and the
// home-page copy. The theme's layouts, components and design remain upstream.
// Forks can therefore be selected with --theme and --theme-repo just like a
// Zola theme, while incompatible starters fail with a useful error.
func (a *Astro) configureAstroPaper(path string, fresh bool) error {
	required := []string{astroThemeConfigFile, astroConfigFile, filepath.Join("src", "pages", "index.astro")}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			return errors.Wrapf(err, "astro theme %q is not AstroPaper-compatible (missing %s)", a.Theme, name)
		}
	}

	if fresh {
		if err := os.RemoveAll(filepath.Join(path, astroPostsDir)); err != nil {
			return errors.Wrap(err, "failed to remove AstroPaper example posts")
		}
	}
	if err := os.MkdirAll(filepath.Join(path, astroPostsDir), 0755); err != nil {
		return errors.Wrap(err, "failed to create Astro post directory")
	}

	if err := os.WriteFile(filepath.Join(path, astroThemeConfigFile), []byte(a.themeConfig()), 0644); err != nil {
		return errors.Wrap(err, "failed to write AstroPaper config")
	}
	if err := a.configureAstro(path); err != nil {
		return err
	}
	if err := a.configurePackage(path); err != nil {
		return err
	}
	if err := a.configureHome(path); err != nil {
		return err
	}
	if err := a.configureAbout(path); err != nil {
		return err
	}
	return a.configureKatexStyles(path)
}

func (a *Astro) themeConfig() string {
	siteURL := a.siteURL()
	socials := "[]"
	if a.Repository != "" {
		socials = fmt.Sprintf(`[{ name: "github", url: %s }]`, jsonString(a.Repository))
	}
	return fmt.Sprintf(`import { defineAstroPaperConfig } from "./src/types/config";

export default defineAstroPaperConfig({
  site: {
    url: %s,
    title: %s,
    description: %s,
    author: %s,
    lang: "en",
    timezone: "UTC",
    dir: "ltr",
  },
  posts: { perPage: 10, perIndex: 10 },
  features: {
    lightAndDarkMode: true,
    dynamicOgImage: true,
    showArchives: true,
    showBackButton: true,
    editPost: { enabled: false },
    search: "pagefind",
  },
  socials: %s,
  shareLinks: [],
});
`, jsonString(siteURL), jsonString(a.Title), jsonString(a.Description), jsonString(a.Author), socials)
}

func (a *Astro) configureAstro(path string) error {
	name := filepath.Join(path, astroConfigFile)
	content, err := os.ReadFile(name)
	if err != nil {
		return errors.Wrap(err, "failed to read Astro config")
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	filtered := lines[:0]
	for _, line := range lines {
		if !strings.Contains(line, "// isite:base") && !strings.Contains(line, "// isite:katex") {
			filtered = append(filtered, line)
		}
	}
	text = strings.Join(filtered, "\n")
	text = strings.ReplaceAll(text, "remarkPlugins: [remarkMath, ", "remarkPlugins: [")
	text = strings.ReplaceAll(text, "rehypePlugins: [rehypeCallouts, rehypeKatex]", "rehypePlugins: [rehypeCallouts]")

	marker := "export default defineConfig({"
	if !strings.Contains(text, marker) {
		return errors.New("Astro theme config does not contain defineConfig")
	}
	_, base := astroDeployment(a.BaseURL)
	text = strings.Replace(text, marker, marker+"\n  base: "+jsonString(base)+", // isite:base", 1)
	if a.Katex {
		text = strings.Replace(text, "import {", `import remarkMath from "remark-math"; // isite:katex
import rehypeKatex from "rehype-katex"; // isite:katex
import {`, 1)
		text = strings.Replace(text, "remarkPlugins: [", "remarkPlugins: [remarkMath, ", 1)
		text = strings.Replace(text, "rehypePlugins: [rehypeCallouts]", "rehypePlugins: [rehypeCallouts, rehypeKatex]", 1)
	}
	if err := os.WriteFile(name, []byte(text), 0644); err != nil {
		return errors.Wrap(err, "failed to write Astro config")
	}
	return nil
}

func (a *Astro) configurePackage(path string) error {
	name := filepath.Join(path, "package.json")
	content, err := os.ReadFile(name)
	if err != nil {
		return errors.Wrap(err, "failed to read Astro package.json")
	}
	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		return errors.Wrap(err, "failed to parse Astro package.json")
	}
	dependencies, ok := manifest["dependencies"].(map[string]any)
	if !ok {
		dependencies = map[string]any{}
		manifest["dependencies"] = dependencies
	}
	for _, dependency := range []string{"katex", "rehype-katex", "remark-math"} {
		delete(dependencies, dependency)
	}
	if a.Katex {
		dependencies["katex"] = "^0.16.22"
		dependencies["rehype-katex"] = "^7.0.1"
		dependencies["remark-math"] = "^6.0.0"
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to encode Astro package.json")
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(name, encoded, 0644); err != nil {
		return errors.Wrap(err, "failed to write Astro package.json")
	}
	return nil
}

func (a *Astro) configureHome(path string) error {
	name := filepath.Join(path, "src", "pages", "index.astro")
	content, err := os.ReadFile(name)
	if err != nil {
		return errors.Wrap(err, "failed to read AstroPaper home page")
	}
	text := string(content)
	start := strings.Index(text, `    <section id="hero"`)
	end := strings.Index(text, "    {\n      featuredPosts")
	if start < 0 || end <= start {
		return errors.New("AstroPaper home page structure is not supported by this isite version")
	}
	rss := ""
	if a.Feed {
		rss = `
      <a target="_blank" rel="noopener noreferrer" href={import.meta.env.BASE_URL.replace(/\/?$/, "/") + "rss.xml"} class="inline-block" aria-label="RSS Feed" title="RSS Feed">
        <IconRss width={20} height={20} class="stroke-accent scale-125 stroke-3 rtl:-rotate-90" />
        <span class="sr-only">RSS Feed</span>
      </a>`
	}
	hero := `    <section id="hero" class="border-border border-b pt-8 pb-6">
      <h1 class="my-4 inline-block text-4xl font-bold sm:my-8 sm:text-5xl">{config.site.title}</h1>` + rss + `
      <p>{config.site.description}</p>
      {socials.length > 0 && (
        <div class="mt-4 flex max-sm:flex-col sm:items-center">
          <div class="me-2 mb-1 whitespace-nowrap sm:mb-0">{t.home.socialLinks}:</div>
          <Socials />
        </div>
      )}
    </section>

`
	text = text[:start] + hero + text[end:]
	if err := os.WriteFile(name, []byte(text), 0644); err != nil {
		return errors.Wrap(err, "failed to write AstroPaper home page")
	}
	return a.configureRSSPage(path)
}

func (a *Astro) configureRSSPage(path string) error {
	rssPage := filepath.Join(path, "src", "pages", "rss.xml.ts")
	backup := filepath.Join(path, ".isite", "astro-paper", "rss.xml.ts")
	if a.Feed {
		if _, err := os.Stat(rssPage); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to stat Astro RSS page")
		}
		content, err := os.ReadFile(backup)
		if err != nil {
			return errors.Wrap(err, "failed to restore Astro RSS page; regenerate into an empty output directory")
		}
		if err := os.WriteFile(rssPage, content, 0644); err != nil {
			return errors.Wrap(err, "failed to restore Astro RSS page")
		}
		return nil
	}

	content, err := os.ReadFile(rssPage)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to read disabled Astro RSS page")
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0755); err != nil {
		return errors.Wrap(err, "failed to create isite theme state directory")
	}
	if err := os.WriteFile(backup, content, 0644); err != nil {
		return errors.Wrap(err, "failed to preserve disabled Astro RSS page")
	}
	if err := os.Remove(rssPage); err != nil {
		return errors.Wrap(err, "failed to remove disabled Astro RSS page")
	}
	return nil
}

func (a *Astro) configureAbout(path string) error {
	dir := filepath.Join(path, "src", "content", "pages")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "failed to create Astro pages directory")
	}
	body := a.Description
	if a.Repository != "" {
		body += fmt.Sprintf("\n\nPosts on this site are generated from [%s issues](%s/issues).", a.Title, a.Repository)
	}
	content := fmt.Sprintf("---\ntitle: %s\ndescription: %s\n---\n\n%s\n", jsonString("About "+a.Title), jsonString(a.Description), body)
	if err := os.WriteFile(filepath.Join(dir, "about.md"), []byte(content), 0644); err != nil {
		return errors.Wrap(err, "failed to write Astro about page")
	}
	return nil
}

func (a *Astro) configureKatexStyles(path string) error {
	name := filepath.Join(path, "src", "styles", "global.css")
	content, err := os.ReadFile(name)
	if err != nil {
		return errors.Wrap(err, "failed to read Astro global styles")
	}
	line := `@import "katex/dist/katex.min.css"; /* isite:katex */` + "\n"
	text := strings.ReplaceAll(string(content), line, "")
	if a.Katex {
		text = line + text
	}
	if err := os.WriteFile(name, []byte(text), 0644); err != nil {
		return errors.Wrap(err, "failed to write Astro global styles")
	}
	return nil
}

func (a *Astro) generatePosts(path string, issues []models.Issue, fresh bool) error {
	dir := filepath.Join(path, astroPostsDir)
	if !fresh {
		matches, err := filepath.Glob(filepath.Join(dir, "issue-*.md"))
		if err != nil {
			return errors.Wrap(err, "failed to list previously generated Astro posts")
		}
		for _, name := range matches {
			if err := os.Remove(name); err != nil {
				return errors.Wrapf(err, "failed to replace generated Astro post %s", name)
			}
		}
	}
	for _, issue := range issues {
		content := a.postContent(issue)
		name := filepath.Join(dir, fmt.Sprintf("issue-%d.md", issue.Number))
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			return errors.Wrapf(err, "failed to write Astro post for issue #%d", issue.Number)
		}
	}
	return nil
}

func (a *Astro) postContent(issue models.Issue) string {
	tags := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		tags = append(tags, label.Name)
	}
	var body bytes.Buffer
	fmt.Fprintf(&body, "---\nauthor: %s\npubDatetime: %s\nmodDatetime: %s\ntitle: %s\ntags: %s\ndescription: %s\ncanonicalURL: %s\n---\n\n",
		jsonString(issue.User.Login), issue.CreatedAt, issue.UpdatedAt, jsonString(issue.Title), jsonValue(tags), jsonString(issue.Title), jsonString(issue.URL))
	if issue.URL != "" {
		fmt.Fprintf(&body, "> Originally published as [GitHub issue #%d](%s).\n\n", issue.Number, issue.URL)
	}
	body.WriteString(issue.Body)
	body.WriteString("\n")
	writeReactions(&body, issue.Reactions)
	if len(issue.Comments) > 0 {
		body.WriteString("\n## Comments\n")
		for _, comment := range issue.Comments {
			fmt.Fprintf(&body, "\n### %s\n\n", comment.User.Login)
			if comment.HTMLURL != "" {
				fmt.Fprintf(&body, "[View comment](%s) · %s\n\n", comment.HTMLURL, comment.UpdatedAt)
			}
			body.WriteString(comment.Body)
			body.WriteString("\n")
		}
	}
	return body.String()
}

func writeReactions(body *bytes.Buffer, reactions models.Reactions) {
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
	if len(parts) > 0 {
		body.WriteString("\n## Reactions\n\n")
		body.WriteString(strings.Join(parts, " · "))
		body.WriteString("\n")
	}
}

func (a *Astro) siteURL() string {
	value := strings.TrimSpace(a.BaseURL)
	parsed, err := url.Parse(value)
	if err == nil && parsed.IsAbs() && parsed.Host != "" {
		return strings.TrimSuffix(value, "/") + "/"
	}
	_, base := astroDeployment(value)
	return "http://localhost:4321" + strings.TrimSuffix(base, "/") + "/"
}

func astroDeployment(baseURL string) (site, base string) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", "/"
	}
	parsed, err := url.Parse(baseURL)
	if err == nil && parsed.IsAbs() && parsed.Host != "" {
		site = parsed.Scheme + "://" + parsed.Host
		base = parsed.EscapedPath()
	} else {
		base = baseURL
	}
	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if base != "/" {
		base = strings.TrimSuffix(base, "/")
	}
	return site, base
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func jsonValue(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
