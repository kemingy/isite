package ssg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kemingy/isite/pkg/models"
)

func TestAstroDeployment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		baseURL string
		site    string
		base    string
	}{
		{name: "default", baseURL: "/", base: "/"},
		{name: "path", baseURL: "docs/", base: "/docs"},
		{name: "root URL", baseURL: "https://example.com", site: "https://example.com", base: "/"},
		{name: "GitHub Pages", baseURL: "https://example.github.io/repository/", site: "https://example.github.io", base: "/repository"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			site, base := astroDeployment(test.baseURL)
			if site != test.site || base != test.base {
				t.Fatalf("astroDeployment(%q) = (%q, %q), want (%q, %q)", test.baseURL, site, base, test.site, test.base)
			}
		})
	}
}

func TestAstroGenerate(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	cmd := &models.Command{
		Engine: "astro", Title: `A "quoted" title`, BaseURL: "https://example.github.io/notes/", Feed: true, Katex: true,
	}
	meta := &models.Repository{Description: "Notes from issues"}
	generator := NewAstro(cmd, meta)
	issue := testAstroIssue()

	if err := generator.Generate([]models.Issue{issue}, output); err != nil {
		t.Fatal(err)
	}

	assertFileContains(t, filepath.Join(output, "astro.config.mjs"),
		`site: "https://example.github.io"`, `base: "/notes"`, "optimizeImages", "remarkMath", "rehypeKatex")
	assertFileContains(t, filepath.Join(output, "package.json"),
		`"astro": "^7.1.1"`, `"@astrojs/rss": "^4.0.19"`, `"@astrojs/markdown-remark": "^7.2.1"`, `"katex": "^0.18.1"`)
	assertFileContains(t, filepath.Join(output, "src", "lib", "site.ts"),
		`export const SITE_TITLE = "A \"quoted\" title"`, `export const FEED = true`)
	assertFileContains(t, filepath.Join(output, "src", "content", "issues", "issue-42.md"),
		`"title": "Front matter: \"safe\""`, `"content": "A multiline comment\nwith --- inside."`, "# Markdown body")
	assertFileContains(t, filepath.Join(output, "src", "pages", "rss.xml.js"), "@astrojs/rss")
	assertFileContains(t, filepath.Join(output, "src", "layouts", "Base.astro"), "katex/dist/katex.min.css")
	assertFileContains(t, filepath.Join(output, "src", "pages", "issue-[number].astro"),
		`url.searchParams.set("s", "64")`)
	assertFileContains(t, filepath.Join(output, "src", "styles", "global.css"), `content-visibility: auto`)
	assertFileContains(t, filepath.Join(output, "src", "lib", "image-optimizer.mjs"),
		`"astro:build:done"`, `addImageAttributes(tag, lazy)`)
}

func TestAstroGenerateWithoutOptionalFeatures(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	staleRSS := filepath.Join(output, "src", "pages", "rss.xml.js")
	if err := os.MkdirAll(filepath.Dir(staleRSS), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleRSS, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	generator := NewAstro(&models.Command{Title: "Notes", BaseURL: "/", Feed: false}, nil)
	if err := generator.Generate(nil, output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staleRSS); !os.IsNotExist(err) {
		t.Fatalf("RSS page should not exist when feed generation is disabled: %v", err)
	}
	packageJSON, err := os.ReadFile(filepath.Join(output, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"@astrojs/rss", "@astrojs/markdown-remark", "remark-math", "rehype-katex", "katex"} {
		if strings.Contains(string(packageJSON), unwanted) {
			t.Fatalf("package.json unexpectedly contains %q", unwanted)
		}
	}
}

func TestAstroLoadsExistingEvenMenu(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	configPath := filepath.Join(output, "config.toml")
	config := `[extra]
even_title = "A personal blog"
even_menu = [
  {url = "$BASE_URL", name = "Home"},
  {url = "$BASE_URL/tags/top/", name = "Top"},
  {url = "$BASE_URL/issue-42/", name = "About"},
]
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	generator := NewAstro(&models.Command{Title: "Notes", Config: configPath}, nil)
	if err := generator.Generate([]models.Issue{testAstroIssue()}, output); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(output, "src", "lib", "site.ts"),
		`SITE_TITLE = "A personal blog"`, `"url":"tags/top/","name":"Top"`, `"url":"issue-42/","name":"About"`)
}

func TestAstroRejectsZolaThemeFlags(t *testing.T) {
	t.Parallel()
	generator := NewAstro(&models.Command{Title: "Notes", Theme: "even", ThemeRepo: "kemingy/even"}, nil)
	err := generator.Generate(nil, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "only supported by the zola engine") {
		t.Fatalf("expected a zola-only theme error, got %v", err)
	}
}

func TestAstroGeneratedProjectBuilds(t *testing.T) {
	if os.Getenv("ISITE_ASTRO_INTEGRATION") == "" {
		t.Skip("set ISITE_ASTRO_INTEGRATION=1 to install dependencies and build the generated Astro project")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}

	output := t.TempDir()
	generator := NewAstro(&models.Command{
		Title: "Integration Notes", BaseURL: "https://example.github.io/notes", Feed: true, Katex: true,
	}, &models.Repository{Description: "An integration build"})
	issues := make([]models.Issue, 11)
	for index := range issues {
		issues[index] = testAstroIssue()
		issues[index].Number = 42 + index
		issues[index].Title = "Integration issue " + string(rune('A'+index))
		issues[index].URL = "https://github.com/example/notes/issues/" + strconv.Itoa(42+index)
	}
	if err := generator.Generate(issues, output); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"install", "--no-audit", "--no-fund"}, {"run", "build"}} {
		command := exec.Command("npm", args...)
		command.Dir = output
		if result, err := command.CombinedOutput(); err != nil {
			t.Fatalf("npm %s failed: %v\n%s", strings.Join(args, " "), err, result)
		}
	}
	for _, name := range []string{"index.html", "page/2/index.html", "issue-42/index.html", "tags/index.html", "tags/astro/index.html", "tags/top/index.html", "rss.xml"} {
		if _, err := os.Stat(filepath.Join(output, "dist", filepath.FromSlash(name))); err != nil {
			t.Fatalf("expected build output %s: %v", name, err)
		}
	}
	assertFileContains(t, filepath.Join(output, "dist", "index.html"), `/notes/issue-42/`, `/notes/page/2/`, `read the source issue`, `Read more...`)
	assertFileContains(t, filepath.Join(output, "dist", "issue-42", "index.html"),
		`/notes/tags/astro/`, `loading="lazy"`, `decoding="async"`, `s=64`)
	assertFileContains(t, filepath.Join(output, "dist", "rss.xml"), `https://example.github.io/notes/issue-42/`)
}

func testAstroIssue() models.Issue {
	return models.Issue{
		Number: 42, Title: `Front matter: "safe"`, URL: "https://github.com/example/notes/issues/42",
		Body: "# Markdown body\n\nInline math: $x^2$.\n\n" + strings.Repeat("Long article text. ", 30) +
			"\n\n![An example](https://example.com/image.png)\n\n<img src=\"https://example.com/raw.png\" alt=\"Raw example\">",
		CreatedAt: "2026-07-19T12:00:00Z", UpdatedAt: "2026-07-20T12:00:00Z",
		User:      models.User{Login: "author", URL: "https://github.com/author", AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4"},
		Labels:    []models.Label{{Name: "astro"}, {Name: "Top"}},
		Reactions: models.Reactions{ThumbUp: 3, Heart: 2},
		Comments: []models.Comment{{
			HTMLURL: "https://github.com/example/notes/issues/42#issuecomment-1",
			User:    models.User{Login: "reader", AvatarURL: "https://avatars.githubusercontent.com/u/2?v=4"},
			Body:    "A multiline comment\nwith --- inside.", UpdatedAt: "2026-07-20T13:00:00Z",
		}},
	}
}

func assertFileContains(t *testing.T, name string, parts ...string) {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range parts {
		if !strings.Contains(string(content), part) {
			t.Errorf("%s does not contain %q", name, part)
		}
	}
}
