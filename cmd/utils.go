package cmd

import (
	"errors"
	"fmt"
	"os"
)

var (
	ProvidedFileError error = errors.New("provided file not directory")
)

func isDirectoryExists(path string) (bool, error) {
	stats, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("Fail get stats: %v", err)
	}

	if !stats.IsDir() {
		return true, ProvidedFileError
	}
	return true, nil
}
