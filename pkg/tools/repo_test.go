package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneThemeReusesExistingDirectory(t *testing.T) {
	t.Parallel()
	path := t.TempDir()

	cloned, err := CloneTheme("example/theme", path, "")
	if err != nil || cloned {
		t.Fatalf("CloneTheme without markers = (%t, %v), want (false, nil)", cloned, err)
	}

	if err := os.WriteFile(filepath.Join(path, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cloned, err = CloneTheme("example/theme", path, "", "package.json")
	if err != nil || cloned {
		t.Fatalf("CloneTheme with existing marker = (%t, %v), want (false, nil)", cloned, err)
	}
}

func TestCloneThemeRejectsNonemptyDirectoryWithoutMarker(t *testing.T) {
	t.Parallel()
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "unrelated.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	cloned, err := CloneTheme("example/theme", path, "", "package.json")
	if cloned || err == nil || !strings.Contains(err.Error(), "missing required markers") {
		t.Fatalf("CloneTheme = (%t, %v), want a missing marker error", cloned, err)
	}
}
