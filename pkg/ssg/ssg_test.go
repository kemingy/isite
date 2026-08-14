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

func TestSanitizeCommentBody(t *testing.T) {
	body := `<script>alert("xss")</script><strong>safe</strong><a href="javascript:alert(1)">bad</a><a href="https://example.com">link</a>`
	sanitized := renderAndSanitizeCommentBody(body)
	if strings.Contains(sanitized, "<script") || strings.Contains(sanitized, "javascript:") {
		t.Fatalf("comment HTML was not sanitized: %s", sanitized)
	}
	if !strings.Contains(sanitized, "<strong>safe</strong>") {
		t.Fatalf("safe comment formatting was removed: %s", sanitized)
	}
	if !strings.Contains(sanitized, `rel="nofollow`) || !strings.Contains(sanitized, `target="_blank"`) {
		t.Fatalf("external comment links were not hardened: %s", sanitized)
	}
	supported := renderAndSanitizeCommentBody(`<details><summary>More</summary><sub>x</sub><p style="text-align: center">c</p></details>`)
	for _, element := range []string{"<details>", "<summary>More</summary>", "<sub>x</sub>", `<p style="text-align: center">c</p>`} {
		if !strings.Contains(supported, element) {
			t.Errorf("UGC policy removed supported comment element %q", element)
		}
	}
}
