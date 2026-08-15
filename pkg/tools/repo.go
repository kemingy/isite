package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const ThemeCacheTTL = 7 * 24 * time.Hour

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
			if revision != "" {
				if err := checkoutExistingThemeRevision(path, revision); err != nil {
					return false, err
				}
			}
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
			if revision != "" {
				if err := checkoutExistingThemeRevision(path, revision); err != nil {
					return false, err
				}
			}
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
		if err := checkoutThemeRevision(cloned, revision); err != nil {
			return false, err
		}
	}
	return true, nil
}

func checkoutExistingThemeRevision(path, revision string) error {
	existing, err := git.PlainOpen(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open existing theme repository %s for revision %s", path, revision)
	}
	return checkoutThemeRevision(existing, revision)
}

func checkoutThemeRevision(repo *git.Repository, revision string) error {
	resolved, err := resolveThemeRevision(repo, revision)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve theme revision %q", revision)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "failed to access theme worktree")
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Hash: *resolved}); err != nil {
		return errors.Wrapf(err, "failed to checkout theme revision %q", revision)
	}
	return nil
}

func resolveThemeRevision(repo *git.Repository, revision string) (*plumbing.Hash, error) {
	resolved, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err == nil {
		return resolved, nil
	}
	// Clone fetches non-default branches as refs/remotes/origin/<branch>.
	// go-git's shorthand resolver does not search remote-tracking refs, so
	// try the equivalent remote ref before reporting the original failure.
	return repo.ResolveRevision(plumbing.Revision("refs/remotes/origin/" + revision))
}

func CloneThemeCached(repo, path, revision string, markers ...string) (bool, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(path); err != nil {
				return false, fmt.Errorf("failed to replace cached theme link %s: %w", path, err)
			}
		} else {
			return CloneTheme(repo, path, revision, markers...)
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to stat theme path %s: %w", path, err)
	}

	cacheRoot, err := themeCacheRoot()
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		return false, fmt.Errorf("failed to create theme cache %s: %w", cacheRoot, err)
	}
	cachePath := filepath.Join(cacheRoot, themeCacheKey(repo, revision))
	cloned := false
	if cacheIsUsable(cachePath, markers) {
		if err := os.Chtimes(cachePath, time.Now(), time.Now()); err != nil {
			return false, fmt.Errorf("failed to refresh theme cache %s: %w", cachePath, err)
		}
	} else {
		if err := os.RemoveAll(cachePath); err != nil {
			return false, fmt.Errorf("failed to remove stale theme cache %s: %w", cachePath, err)
		}
		temporary, err := os.MkdirTemp(cacheRoot, ".theme-")
		if err != nil {
			return false, fmt.Errorf("failed to create temporary theme cache: %w", err)
		}
		defer os.RemoveAll(temporary)
		if _, err := CloneTheme(repo, temporary, revision, markers...); err != nil {
			return false, err
		}
		cloned = true
		if err := os.Rename(temporary, cachePath); err != nil {
			return false, fmt.Errorf("failed to store theme cache %s: %w", cachePath, err)
		}
	}
	if err := os.Symlink(cachePath, path); err != nil {
		return false, fmt.Errorf("failed to link cached theme %s to %s: %w", cachePath, path, err)
	}
	return cloned, nil
}

func PruneThemeCache() (int, error) {
	root, err := themeCacheRoot()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read theme cache %s: %w", root, err)
	}
	removed := 0
	cutoff := time.Now().Add(-ThemeCacheTTL)
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) != sha256.Size*2 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return removed, fmt.Errorf("failed to stat theme cache %s: %w", entry.Name(), err)
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return removed, fmt.Errorf("failed to remove expired theme cache %s: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}

func PruneOutput(path string) (bool, error) {
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) || clean == "" {
		return false, fmt.Errorf("refusing to remove unsafe output path %q", path)
	}
	info, err := os.Stat(clean)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat output path %s: %w", clean, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("output path %s is not a directory", clean)
	}
	if err := os.RemoveAll(clean); err != nil {
		return false, fmt.Errorf("failed to remove output path %s: %w", clean, err)
	}
	return true, nil
}

func themeCacheRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find the user home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "isite", "themes"), nil
}

func themeCacheKey(repo, revision string) string {
	sum := sha256.Sum256([]byte(repo + "\x00" + revision))
	return hex.EncodeToString(sum[:])
}

func cacheIsUsable(path string, markers []string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() || time.Since(info.ModTime()) >= ThemeCacheTTL {
		return false
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(path, marker)); err != nil {
			return false
		}
	}
	return true
}
