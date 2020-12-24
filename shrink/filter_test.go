package shrink

import (
	"fmt"
	"testing"

	. "github.com/icecream78/node_shrinker/walker"
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
			assert.Equal(t, tc.want, filter.isIncludeName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestRemoveFileNameFunc(t *testing.T) {
	includeNames := []string{
		"file",
		"/a/b/c/file",
	}

	filter := NewFilter(includeNames, []string{}, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test by relative path added by name", "file", true},
		{"Test by absolute path added by name", "/a/b/c/file", true},
		{"Test by relative path not added by name", "file2", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isIncludeName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestExcludeNameFunc(t *testing.T) {
	excludeNames := []string{
		"file",
		"/a/b/c/file",
	}

	filter := NewFilter([]string{}, excludeNames, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test by relative path added by name", "file", true},
		{"Test by absolute path added by name", "/a/b/c/file", true},
		{"Test by relative path not added by name", "file2", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isExcludeName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestIncludeExtensionFunc(t *testing.T) {
	removeFileExt := []string{
		".js",
		"js",
	}

	filter := NewFilter([]string{}, []string{}, removeFileExt)

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test by filename + extension", "test.js", true},
		{"Test by . + filename + extension", ".file.js", true},
		{"Test by . + filename as extension + extension", ".js.js", true},
		{"Test by single . + extension", ".js", false}, // hidden file
		{"Test by single extension name", "js", false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isIncludeExt(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestExcludeRegNameFunc(t *testing.T) {
	excludes := []string{
		"rem*",
	}
	filter := NewFilter([]string{}, excludes, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test excluded file by relative path", "remove", true},
		{"Test excluded file by absolute path", "/a/b/remove/file", true},
		{"Test not excluded file", "file2", false},
		{"Test exclude file by regexp", "remove.js", true},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isExcludeRegName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestIncludeRegNameFunc(t *testing.T) {
	includes := []string{
		"rem*",
	}
	filter := NewFilter(includes, []string{}, []string{})

	testCases := []struct {
		alias string
		name  string
		want  bool
	}{
		{"Test include file by relative path", "remove", true},
		{"Test include file by absolute path", "/a/b/remove/file", true},
		{"Test not include file", "file2", false},
		{"Test include file by regexp", "remove.js", true},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			assert.Equal(t, tc.want, filter.isIncludeRegName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

type fileTestStub struct {
	name      string
	isRegular bool
}

func newFileTestStub(name string, isFile bool) *fileTestStub {
	return &fileTestStub{
		name:      name,
		isRegular: isFile,
	}
}

func (s *fileTestStub) Name() string {
	return s.name
}

func (s *fileTestStub) IsDir() bool {
	return !s.isRegular
}

func (s *fileTestStub) IsRegular() bool {
	return s.isRegular
}

func TestCheckFunc(t *testing.T) {
	includes := []string{
		"file1",
		"nam*",
	}
	extensions := []string{
		".js",
	}
	exlcudes := []string{
		"file2",
		"sur*",
	}

	filter := NewFilter(includes, exlcudes, extensions)

	testCases := []struct {
		alias       string
		input       *fileTestStub
		wantedBool  bool
		wantedError error
	}{
		{alias: "Include regular file name", input: newFileTestStub("file1", true), wantedBool: true, wantedError: nil},
		{alias: "Include regexp file name", input: newFileTestStub("name.txt", true), wantedBool: true, wantedError: nil},
		{alias: "Include extension file name", input: newFileTestStub("script.js", true), wantedBool: true, wantedError: nil},
		{alias: "Test dir with extension on the end of file name", input: newFileTestStub("scripts.js", false), wantedBool: false, wantedError: NotProcessError},
		{alias: "Exclude regular file name", input: newFileTestStub("file2", true), wantedBool: false, wantedError: ExcludeError},
		{alias: "Exclude regexp file name", input: newFileTestStub("surname.txt", true), wantedBool: false, wantedError: ExcludeError},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			wanted, err := filter.Check(tc.input)

			assert.Equal(t, tc.wantedBool, wanted, fmt.Sprintf("Expected bool: %v, got bool: %v", tc.wantedBool, wanted))
			assert.Equal(t, tc.wantedError, err, fmt.Sprintf("Expected error: %v, got bool: %v", tc.wantedError, err))
		})
	}
}
