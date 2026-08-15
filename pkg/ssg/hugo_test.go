package ssg

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kemingy/isite/pkg/models"
)

func TestNewHugoUsesPaperModByDefault(t *testing.T) {
	generator := NewHugo(&models.Command{Title: testTitle}, nil)
	if generator.ThemeName != hugoDefaultTheme || generator.ThemeRepo != hugoDefaultThemeRepo {
		t.Fatalf("default theme = %q (%q), want %q (%q)", generator.ThemeName, generator.ThemeRepo, hugoDefaultTheme, hugoDefaultThemeRepo)
	}
}

func TestHugoGenerate(t *testing.T) {
	output := t.TempDir()
	prepareHugoTestTheme(t, output)
	generator := NewHugo(&models.Command{
		Title: "A Hugo Notes \"site\"", BaseURL: "https://example.com/notes", Theme: "paper", ThemeRepo: "example/paper", Feed: true,
	}, &models.Repository{Description: "A description"})
	if err := generator.Generate([]models.Issue{hugoTestIssue()}, output); err != nil {
		t.Fatal(err)
	}
	assertHugoGeneratedFiles(t, output)
}

func prepareHugoTestTheme(t *testing.T, output string) {
	t.Helper()
	themeDir := filepath.Join(output, "themes", "paper")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "theme.toml"), []byte("name = 'PaperMod'\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func hugoTestIssue() models.Issue {
	return models.Issue{
		Number: 42, Title: "A quoted title", URL: "https://github.com/example/notes/issues/42",
		Body: "# Markdown body", CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-03T03:04:05Z",
		User: models.User{Login: "author"}, Labels: []models.Label{{Name: "hugo"}},
		Reactions: models.Reactions{ThumbUp: 3, Heart: 2},
		Comments:  []models.Comment{{User: models.User{Login: "reader"}, HTMLURL: "https://example.com/comment", UpdatedAt: "2026-01-04", Body: "> A quoted comment\n\n```toml\ntitle = \"quote\"\n```"}},
	}
}

func assertHugoGeneratedFiles(t *testing.T, output string) {
	t.Helper()
	config := readHugoFile(t, filepath.Join(output, "hugo.toml"))
	post := readHugoFile(t, filepath.Join(output, hugoContentDir, "issue-42.md"))
	ogImage := readHugoFile(t, filepath.Join(output, hugoOGDir, "issue-42.svg"))
	for _, text := range []string{"theme = '''paper'''", `env = "production"`, `unsafe = false`, "description = '''A description'''", "disableSpecial1stPost = true", `name = "Search"`, `home = ["HTML", "JSON", "RSS"]`} {
		if !strings.Contains(config, text) {
			t.Errorf("Hugo config does not contain %q:\n%s", text, config)
		}
	}
	for _, text := range []string{"title = '''A quoted title'''", "tags = ['''hugo''']", "image = \"/images/og/issue-42.svg\"", "commentCount = 1", `contentKey = "issue-42"`} {
		if !strings.Contains(post, text) {
			t.Errorf("Hugo post does not contain %q:\n%s", text, post)
		}
	}
	assertFileContains(t, filepath.Join(output, hugoEngagementPartial), "isite-reactions", "isite-comments", "Read-only mirror", "this GitHub issue", "safeHTML", "isite-comment-body md-content", "site.Data.comments", ".body", "site.Data.posts")
	assertFileContains(t, filepath.Join(output, hugoTOCPartial), "site.Data.posts", "tocHTML", "safeHTML")
	commentsJSON := readHugoFile(t, filepath.Join(output, hugoCommentsDataDir, "issue-42.json"))
	if !strings.Contains(commentsJSON, "A quoted comment") || !strings.Contains(commentsJSON, `\u003cblockquote\u003e`) || !strings.Contains(commentsJSON, `class=\"chroma\"`) {
		t.Errorf("comments JSON does not contain sanitized HTML: %s", commentsJSON)
	}
	postData := readHugoFile(t, filepath.Join(output, hugoPostsDataDir, "issue-42.json"))
	for _, text := range []string{"bodyHTML", "tocHTML", `id=\"markdown-body\"`, `href=\"#markdown-body\"`} {
		if !strings.Contains(postData, text) {
			t.Errorf("Hugo post data does not contain %q: %s", text, postData)
		}
	}
	if strings.Contains(post, "## Comments") || strings.Contains(post, "## Reactions") {
		t.Fatal("reactions and comments should not be part of the post content")
	}
	if _, err := os.Stat(filepath.Join(output, hugoCommentsDataDir, "issue-42.json")); err != nil {
		t.Fatalf("comments data was not generated: %v", err)
	}
	if !strings.Contains(ogImage, "<svg") || !strings.Contains(ogImage, "A quoted title") {
		t.Errorf("generated OG image does not contain the issue title:\n%s", ogImage)
	}
	var svg struct{}
	if err := xml.Unmarshal([]byte(ogImage), &svg); err != nil {
		t.Fatalf("generated OG image is not valid XML: %v", err)
	}
	for _, name := range []string{hugoAboutFile, hugoSearchFile} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Errorf("expected generated Hugo page %s: %v", name, err)
		}
	}
}

func readHugoFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestOGTitleLinesWrapLongTitles(t *testing.T) {
	lines := ogTitleLines("A very long post title that should be split across multiple lines", 20)
	if len(lines) < 3 {
		t.Fatalf("wrapped title has %d lines, want at least 3: %#v", len(lines), lines)
	}
	for _, line := range lines {
		if len([]rune(line)) > 20 {
			t.Fatalf("wrapped line is too long: %q", line)
		}
	}
}

func TestHugoTOCIncludesOnlyFirstThreeHeadingLevels(t *testing.T) {
	body := renderAndSanitizeMarkdown("# One\n\n## Two\n\n### Three\n\n#### Four")
	toc := hugoTOC(body)
	for _, heading := range []string{"One", "Two", "Three"} {
		if !strings.Contains(toc, ">"+heading+"</a>") {
			t.Errorf("TOC does not contain %q: %s", heading, toc)
		}
	}
	if strings.Contains(toc, "Four") {
		t.Errorf("TOC should not contain h4 headings: %s", toc)
	}
}

func TestHugoCodeBlocksAreHighlighted(t *testing.T) {
	body := renderAndSanitizeMarkdown("```go\nfunc main() {}\n```")
	for _, marker := range []string{`class="chroma"`, `class="kd"`} {
		if !strings.Contains(body, marker) {
			t.Errorf("highlighted body does not contain %q: %s", marker, body)
		}
	}
}
