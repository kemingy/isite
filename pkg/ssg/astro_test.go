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

func TestNewAstroUsesAstroPaperByDefault(t *testing.T) {
	t.Parallel()
	generator := NewAstro(&models.Command{Title: "Notes"}, nil)
	if generator.Theme != astroDefaultTheme || generator.ThemeRepo != astroDefaultThemeRepo {
		t.Fatalf("default theme = %q (%q), want %q (%q)", generator.Theme, generator.ThemeRepo, astroDefaultTheme, astroDefaultThemeRepo)
	}

	custom := NewAstro(&models.Command{Title: "Notes", Theme: "paper-fork", ThemeRepo: "example/paper-fork"}, nil)
	if custom.Theme != "paper-fork" || custom.ThemeRepo != "example/paper-fork" {
		t.Fatalf("custom theme flags were not preserved: %#v", custom)
	}
}

func TestAstroGenerate(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	prepareAstroPaper(t, output)
	cmd := &models.Command{
		Engine: "astro", Title: `A "quoted" title`, BaseURL: "https://example.github.io/notes/", Feed: true, Katex: true,
	}
	meta := &models.Repository{
		Description: "Notes from issues", FullName: "example/notes", Owner: models.User{Login: "owner"},
	}
	generator := NewAstro(cmd, meta)
	issue := testAstroIssue()

	if err := generator.Generate([]models.Issue{issue}, output); err != nil {
		t.Fatal(err)
	}

	assertFileContains(t, filepath.Join(output, "astro.config.ts"),
		`base: "/notes"`, "remarkMath", "rehypeKatex")
	assertFileContains(t, filepath.Join(output, "package.json"),
		`"astro": "^6.4.2"`, `"katex": "^0.16.22"`, `"rehype-katex": "^7.0.1"`, `"remark-math": "^6.0.0"`)
	assertFileContains(t, filepath.Join(output, "astro-paper.config.ts"),
		`url: "https://example.github.io/notes/"`, `title: "A \"quoted\" title"`, `author: "owner"`,
		`https://github.com/example/notes`)
	assertFileContains(t, filepath.Join(output, astroPostsDir, "issue-42.md"),
		`title: "Front matter: \"safe\""`, `tags: ["astro","Top"]`, "# Markdown body",
		"## Reactions", "👍 3 · ❤️ 2", "## Comments", "A multiline comment\nwith --- inside.")
	assertFileContains(t, filepath.Join(output, "src", "pages", "index.astro"),
		"{config.site.title}", "{config.site.description}", "rss.xml", `rel="noopener noreferrer"`)
	assertFileContains(t, filepath.Join(output, "src", "content", "pages", "about.md"),
		"Notes from issues", "https://github.com/example/notes/issues")
	assertFileContains(t, filepath.Join(output, "src", "styles", "global.css"), "katex/dist/katex.min.css")
}

func TestAstroGenerateWithoutOptionalFeatures(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	prepareAstroPaper(t, output)
	generator := NewAstro(&models.Command{Title: "Notes", BaseURL: "/", Feed: true, Katex: true}, nil)
	if err := generator.Generate(nil, output); err != nil {
		t.Fatal(err)
	}

	generator = NewAstro(&models.Command{Title: "Notes", BaseURL: "/", Feed: false, Katex: false}, nil)
	if err := generator.Generate(nil, output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "src", "pages", "rss.xml.ts")); !os.IsNotExist(err) {
		t.Fatalf("RSS page should not exist when feed generation is disabled: %v", err)
	}
	for _, name := range []string{"package.json", "astro.config.ts", filepath.Join("src", "styles", "global.css")} {
		content, err := os.ReadFile(filepath.Join(output, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, unwanted := range []string{"remark-math", "rehype-katex", "katex/dist", "remarkMath", "rehypeKatex"} {
			if strings.Contains(string(content), unwanted) {
				t.Fatalf("%s unexpectedly contains %q", name, unwanted)
			}
		}
	}
	content, err := os.ReadFile(filepath.Join(output, "src", "pages", "index.astro"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "rss.xml") {
		t.Fatal("home page unexpectedly links to disabled RSS feed")
	}

	generator = NewAstro(&models.Command{Title: "Notes", BaseURL: "/", Feed: true, Katex: false}, nil)
	if err := generator.Generate(nil, output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "src", "pages", "rss.xml.ts")); err != nil {
		t.Fatalf("RSS page was not restored when the feed was re-enabled: %v", err)
	}
}

func TestAstroRefreshPreservesNonGeneratedPosts(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	prepareAstroPaper(t, output)
	custom := filepath.Join(output, astroPostsDir, "custom.md")
	stale := filepath.Join(output, astroPostsDir, "issue-99.md")
	if err := os.MkdirAll(filepath.Dir(custom), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(custom, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	generator := NewAstro(&models.Command{Title: "Notes", Feed: true}, nil)
	if err := generator.Generate([]models.Issue{testAstroIssue()}, output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatalf("custom post was not preserved: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale generated post was not removed: %v", err)
	}
}

func TestAstroRejectsIncompatibleTheme(t *testing.T) {
	t.Parallel()
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(output, "package.json"), []byte(`{"dependencies":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	generator := NewAstro(&models.Command{Title: "Notes", Theme: "other", ThemeRepo: "example/other"}, nil)
	err := generator.Generate(nil, output)
	if err == nil || !strings.Contains(err.Error(), "not AstroPaper-compatible") {
		t.Fatalf("expected a theme compatibility error, got %v", err)
	}
}

func TestAstroGeneratedProjectBuilds(t *testing.T) {
	if os.Getenv("ISITE_ASTRO_INTEGRATION") == "" {
		t.Skip("set ISITE_ASTRO_INTEGRATION=1 to clone AstroPaper, install dependencies and build the generated project")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}

	output := filepath.Join(t.TempDir(), "site")
	generator := NewAstro(&models.Command{
		Title: "Integration Notes", BaseURL: "https://example.github.io/notes", Feed: true, Katex: true,
	}, &models.Repository{
		Description: "An integration build", FullName: "example/notes", Owner: models.User{Login: "example"},
	})
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
	for _, name := range []string{
		"index.html", "posts/2/index.html", "posts/issue-42/index.html", "tags/index.html", "tags/astro/index.html", "rss.xml",
	} {
		if _, err := os.Stat(filepath.Join(output, "dist", filepath.FromSlash(name))); err != nil {
			t.Fatalf("expected build output %s: %v", name, err)
		}
	}
	assertFileContains(t, filepath.Join(output, "dist", "index.html"), "Integration Notes", `/notes/posts/issue-42/`)
	assertFileContains(t, filepath.Join(output, "dist", "posts", "issue-42", "index.html"),
		`/notes/tags/astro/`, "GitHub issue #42", "Comments", "Reactions")
	assertFileContains(t, filepath.Join(output, "dist", "rss.xml"), `/notes/posts/issue-42`)
}

func prepareAstroPaper(t *testing.T, output string) {
	t.Helper()
	files := map[string]string{
		"package.json":          `{"dependencies":{"astro":"^6.4.2"}}`,
		"astro-paper.config.ts": `export default {};`,
		"astro.config.ts": `import { defineConfig } from "astro/config";
import rehypeCallouts from "rehype-callouts";
export default defineConfig({
  markdown: {
    processor: {},
    remarkPlugins: [],
    rehypePlugins: [rehypeCallouts],
  },
});`,
		filepath.Join("src", "pages", "index.astro"): `---
import config from "@/config";
const { socials } = config;
---
  <main>
    <section id="hero" class="old">old theme demo</section>

    {
      featuredPosts.length > 0 && <div />
    }
  </main>`,
		filepath.Join("src", "pages", "rss.xml.ts"):  `export function GET() {}`,
		filepath.Join("src", "styles", "global.css"): `body { color: black; }`,
	}
	for name, content := range files {
		fullName := filepath.Join(output, name)
		if err := os.MkdirAll(filepath.Dir(fullName), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullName, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
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
