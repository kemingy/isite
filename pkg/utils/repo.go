package utils

import (
	"fmt"
	"os"

	"github.com/cockroachdb/errors"
	git "github.com/go-git/go-git/v5"
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

func CloneTheme(repo, path string) error {
	exist, err := DirExist(path)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	fmt.Printf("clone the theme from %s to %s\n", repo, path)
	_, err = git.PlainClone(path, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s", repo),
	},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to clone the repo %s", repo)
	}
	return nil
}
