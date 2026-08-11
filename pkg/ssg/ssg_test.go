package ssg

import (
	"strings"
	"testing"

	"github.com/kemingy/isite/pkg/models"
)

func TestGeneratorsRejectEmptyOutput(t *testing.T) {
	t.Parallel()
	generators := map[string]StaticSiteGenerator{
		"astro": NewAstro(&models.Command{Title: "Notes"}, nil),
		"zola":  NewZola(&models.Command{Title: "Notes"}, nil),
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
