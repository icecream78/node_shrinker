package shrink

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveDirNameFunc(t *testing.T) {
	includes := []string{
		"dirname",
		"/a/b/c/dirname",
	}
	filter := NewFilter(includes, []string{}, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test excluded directory by relative path", "dirname", true},
		{"Test excluded directory by absolute path", "/a/b/c/dirname", true},
		{"Test not excluded directory", "dirname2", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isDirToRemove(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestRemoveFileNameFunc(t *testing.T) {
	includeNames := []string{
		"file",
		"/a/b/c/file",
	}
	removeFileExt := []string{
		".js",
		"js",
	}

	filter := NewFilter(includeNames, []string{}, removeFileExt)

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test by relative path added by name", "file", true},
		{"Test by absolute path added by name", "/a/b/c/file", true},
		{"Test by relative path not added by name", "file2", false},
		{"Test by filename + extension", "test.js", true},
		{"Test by . + filename + extension", ".file.js", true},
		{"Test by . + filename as extension + extension", ".js.js", true},
		{"Test by single . + extension", ".js", false}, // hidden file
		{"Test by single extension name", "js", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isFileToRemove(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestExcludeNameFunc(t *testing.T) {
	excludes := []string{
		"file",
		"/a/b/c/file",
	}
	filter := NewFilter([]string{}, excludes, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test excluded file by relative path", "file", true},
		{"Test excluded file by absolute path", "/a/b/c/file", true},
		{"Test not excluded file", "file2", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isExcludeName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}
