package helpers

import (
	"os"
	"path/filepath"
)

func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	return "", os.ErrNotExist
}
