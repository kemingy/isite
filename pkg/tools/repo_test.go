package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func TestCheckoutThemeRevisionReusesExistingCheckout(t *testing.T) {
	path := t.TempDir()
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	writeRevisionFile := func(contents string) plumbing.Hash {
		t.Helper()
		if err := os.WriteFile(filepath.Join(path, "package.json"), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := worktree.Add("package.json"); err != nil {
			t.Fatal(err)
		}
		hash, err := worktree.Commit(contents, &git.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()}})
		if err != nil {
			t.Fatal(err)
		}
		return hash
	}

	first := writeRevisionFile("first")
	second := writeRevisionFile("second")
	if err := checkoutThemeRevision(repo, first.String()); err != nil {
		t.Fatal(err)
	}
	if err := checkoutThemeRevision(repo, second.String()); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(path, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "second" {
		t.Fatalf("existing checkout contents = %q, want second", contents)
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
