package ssg

import (
	"strings"
	"testing"

	"github.com/kemingy/isite/pkg/models"
)

const testTitle = "Notes"

func TestGeneratorsRejectEmptyOutput(t *testing.T) {
	t.Parallel()
	generators := map[string]StaticSiteGenerator{
		engineAstro: NewAstro(&models.Command{Title: testTitle}, nil),
		engineZola:  NewZola(&models.Command{Title: testTitle}, nil),
		engineHugo:  NewHugo(&models.Command{Title: testTitle}, nil),
	}
	for name, generator := range generators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, output := range []string{"", "  \t\n"} {
				err := generator.Generate(nil, output)
				if err == nil || !strings.Contains(err.Error(), "output directory must not be empty") {
					t.Fatalf("Generate with output %q returned %v", output, err)
				}
			}
		})
	}
}

func TestSanitizeMarkdownSourcePreservesMarkdown(t *testing.T) {
	source := "# Heading\n\n```go\nfmt.Println(\"safe\")\n```\n\n<script>alert(1)</script>"
	sanitized := sanitizeMarkdownSource(source)
	if !strings.Contains(sanitized, "# Heading") || !strings.Contains(sanitized, "```go") {
		t.Fatalf("Markdown source was changed unexpectedly: %s", sanitized)
	}
	if strings.Contains(sanitized, "<script") {
		t.Fatalf("unsafe HTML was not removed: %s", sanitized)
	}
	supported := sanitizeMarkdownSource(`<details><summary>More</summary><sub>x</sub><p style="text-align: center">c</p></details>`)
	for _, element := range []string{"<details>", "<summary>More</summary>", "<sub>x</sub>", `<p style="text-align: center">c</p>`} {
		if !strings.Contains(supported, element) {
			t.Errorf("UGC policy removed supported comment element %q", element)
		}
	}
}
