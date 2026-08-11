package pkg

import (
	"strings"
	"testing"

	"github.com/kemingy/isite/pkg/models"
)

func TestWebsiteGenerateRejectsEmptyOutput(t *testing.T) {
	t.Parallel()
	website := &Website{}
	for _, output := range []string{"", "  \t\n"} {
		err := website.Generate(&models.Command{Engine: "zola", Output: output})
		if err == nil || !strings.Contains(err.Error(), "output directory must not be empty") {
			t.Fatalf("Generate with output %q returned %v", output, err)
		}
	}
}
