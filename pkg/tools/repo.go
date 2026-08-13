package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func DirExist(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to stat the path %s", path)
	}
	return info.IsDir(), nil
}

// CloneTheme clones repo into path at revision and reports whether a clone was performed.
// Existing directories are reused immediately when no markers are supplied.
// When markers such as package.json are supplied, every marker must exist;
// otherwise only an empty directory is safe to clone into.
// Revision may be empty to use the repository's default branch.
func CloneTheme(repo, path, revision string, markers ...string) (bool, error) {
	exist, err := DirExist(path)
	if err != nil {
		return false, err
	}
	if exist {
		if len(markers) == 0 {
			return false, nil
		}
		allMarkersExist := true
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(path, marker)); err != nil {
				if !os.IsNotExist(err) {
					return false, errors.Wrapf(err, "failed to stat clone marker %s", marker)
				}
				allMarkersExist = false
			}
		}
		if allMarkersExist {
			return false, nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return false, errors.Wrapf(err, "failed to read the clone directory %s", path)
		}
		if len(entries) > 0 {
			return false, errors.Errorf("clone directory %s is not empty and is missing required markers %v", path, markers)
		}
	}

	fmt.Printf("clone the theme(revision:%s) from %s to %s\n", revision, repo, path)
	cloned, err := git.PlainClone(path, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s", repo),
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to clone the repo %s", repo)
	}
	if revision != "" {
		resolved, err := cloned.ResolveRevision(plumbing.Revision(revision))
		if err != nil {
			return false, errors.Wrapf(err, "failed to resolve theme revision %q", revision)
		}
		worktree, err := cloned.Worktree()
		if err != nil {
			return false, errors.Wrap(err, "failed to access cloned theme worktree")
		}
		if err := worktree.Checkout(&git.CheckoutOptions{Hash: *resolved}); err != nil {
			return false, errors.Wrapf(err, "failed to checkout theme revision %q", revision)
		}
	}
	return true, nil
}
