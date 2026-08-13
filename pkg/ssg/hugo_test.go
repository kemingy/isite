package ssg

import (
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
	themeDir := filepath.Join(output, "themes", "paper")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "theme.toml"), []byte("name = 'PaperMod'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	generator := NewHugo(&models.Command{
		Title: "A Hugo Notes \"site\"", BaseURL: "https://example.com/notes", Theme: "paper", ThemeRepo: "example/paper", Feed: true,
	}, &models.Repository{Description: "A description"})
	issue := models.Issue{
		Number: 42, Title: "A quoted title", URL: "https://github.com/example/notes/issues/42",
		Body: "# Markdown body", CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-03T03:04:05Z",
		User: models.User{Login: "author"}, Labels: []models.Label{{Name: "hugo"}},
		Reactions: models.Reactions{ThumbUp: 3, Heart: 2},
		Comments:  []models.Comment{{User: models.User{Login: "reader"}, HTMLURL: "https://example.com/comment", UpdatedAt: "2026-01-04", Body: "A comment"}},
	}
	if err := generator.Generate([]models.Issue{issue}, output); err != nil {
		t.Fatal(err)
	}
	config := readHugoFile(t, filepath.Join(output, "hugo.toml"))
	post := readHugoFile(t, filepath.Join(output, hugoContentDir, "issue-42.md"))
	ogImage := readHugoFile(t, filepath.Join(output, hugoOGDir, "issue-42.svg"))
	for _, text := range []string{"theme = '''paper'''", "description = '''A description'''", "disableSpecial1stPost = true", `name = "Search"`, `home = ["HTML", "JSON", "RSS"]`} {
		if !strings.Contains(config, text) {
			t.Errorf("Hugo config does not contain %q:\n%s", text, config)
		}
	}
	for _, text := range []string{"title = '''A quoted title'''", "tags = ['''hugo''', ]", "image = \"/images/og/issue-42.svg\"", "# Markdown body", "## Reactions", "👍 3 · ❤️ 2", "## Comments", "A comment"} {
		if !strings.Contains(post, text) {
			t.Errorf("Hugo post does not contain %q:\n%s", text, post)
		}
	}
	if !strings.Contains(ogImage, "<svg") || !strings.Contains(ogImage, "A quoted title") {
		t.Errorf("generated OG image does not contain the issue title:\n%s", ogImage)
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
