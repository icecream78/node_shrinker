package shrink

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/icecream78/node_shrinker/fs"
	"github.com/icecream78/node_shrinker/mocks"

	. "github.com/icecream78/node_shrinker/walker"
	"github.com/stretchr/testify/assert"
)

type walkerStub struct {
	testPathes map[string]string
}

func (w *walkerStub) SetFileStructure(pathes map[string]string) {
	w.testPathes = pathes
}

func (w *walkerStub) Walk(filepath string, callback WalkFunc, errCallback WalkErrFunc) error {
	for path := range w.testPathes {
		callback(path, &FileInfo{})
	}
	return nil
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

func TestStatGrabberFunc(t *testing.T) {
	testCases := []struct {
		alias string
		input []fs.FileStat
		want  fs.FileStat
	}{
		{"Check stats grabber with empty input", []fs.FileStat{}, *fs.NewFileStat("result", "result", 0, 0)},
		{"Put few files for calculations", []fs.FileStat{
			*fs.NewFileStat("1", "1", 1024, 1),
			*fs.NewFileStat("2", "2", 1024, 1),
		}, *fs.NewFileStat("result", "result", 2048, 2)},
	}
	for _, tc := range testCases {
		sh, _ := NewShrinker(&Config{})
		resStatsCh := sh.runStatGrabber()
		t.Run(tc.alias, func(t *testing.T) {
			for _, file := range tc.input {
				sh.statsCh <- file
			}
			close(sh.statsCh)
			stats := <-resStatsCh
			assert.Equal(t, tc.want, stats, fmt.Sprintf("Input: %v", tc.input))
		})
	}
}

type testFileInfo struct {
	name      string
	isDir     bool
	isRegular bool
}

func (tfi testFileInfo) Name() string {
	return tfi.name
}

func (tfi testFileInfo) IsDir() bool {
	return tfi.isDir
}

func (tfi testFileInfo) IsRegular() bool {
	return tfi.isRegular
}

func TestFileFilterExcludeByName(t *testing.T) {
	sh, _ := NewShrinker(&Config{
		ExcludeNames: []string{
			"file1",
		},
		IncludeNames: []string{
			"dirname1",
		},
	})

	fileFullPath := "/file1"
	input := testFileInfo{name: "file1", isDir: false, isRegular: true}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, ExcludeError, err)

	close(sh.removeCh)
	info, ok := <-sh.removeCh

	expectedChannelCloseStatus := false
	var expectedInfo *removeObjInfo = nil

	assert.Equal(t, expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(t, expectedInfo, info)
}

func TestFileFilterIncludeByName(t *testing.T) {
	sh, _ := NewShrinker(&Config{
		IncludeNames: []string{
			"file1",
		},
	})

	fileFullPath := "/file1"
	input := testFileInfo{name: "file1", isDir: false, isRegular: true}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, nil, err)

	close(sh.removeCh)
	info := <-sh.removeCh

	expectedChannelCloseStatus := false
	expectedInfo := &removeObjInfo{filename: "file1", fullpath: "/file1", isDir: false}
	assert.Equal(t, expectedInfo, info)

	_, ok := <-sh.removeCh
	assert.Equal(t, expectedChannelCloseStatus, ok, "channel is not closed")
}

func TestFileFilterExcludeByRegexp(t *testing.T) {
	sh, _ := NewShrinker(&Config{
		ExcludeNames: []string{
			"file1",
			"sc*",
		},
		IncludeNames: []string{
			"dirname1",
		},
	})

	fileFullPath := "/script.1.js"
	input := testFileInfo{name: "script.1.js", isDir: false, isRegular: true}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, ExcludeError, err)

	close(sh.removeCh)
	info, ok := <-sh.removeCh

	expectedChannelCloseStatus := false
	var expectedInfo *removeObjInfo = nil

	assert.Equal(t, expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(t, expectedInfo, info)
}

func TestFileFilterIncludeByRegexp(t *testing.T) {
	sh, _ := NewShrinker(&Config{
		IncludeNames: []string{
			"dirname1",
			"f*",
		},
	})

	fileFullPath := "/file2.ts"
	input := testFileInfo{name: "file2.ts", isDir: false, isRegular: true}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, nil, err)

	info := <-sh.removeCh
	close(sh.removeCh)

	expectedChannelCloseStatus := false
	expectedInfo := &removeObjInfo{isDir: false, filename: "file2.ts", fullpath: "/file2.ts"}
	assert.Equal(t, expectedInfo, info)

	_, ok := <-sh.removeCh
	assert.Equal(t, expectedChannelCloseStatus, ok, "is channel closed")
}

func TestFileFilterNotProcessDir(t *testing.T) {
	sh, _ := NewShrinker(&Config{})

	fileFullPath := "/dirname1"
	input := testFileInfo{name: "dirname1", isDir: true, isRegular: false}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, NotProcessError, err)

	close(sh.removeCh)
	info, ok := <-sh.removeCh

	expectedChannelCloseStatus := false
	var expectedInfo *removeObjInfo = nil

	assert.Equal(t, expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(t, expectedInfo, info)
}

func TestFileFilterRemoveDir(t *testing.T) {
	sh, _ := NewShrinker(&Config{
		IncludeNames: []string{
			"dirname1",
		},
	})

	fileFullPath := "/dirname1"
	input := testFileInfo{name: "dirname1", isDir: true, isRegular: false}

	err := sh.fileFilterCallback(fileFullPath, input)
	assert.Equal(t, SkipDirError, err) // for app logic

	close(sh.removeCh)
	info := <-sh.removeCh

	expectedInfo := &removeObjInfo{isDir: true, filename: "dirname1", fullpath: "/dirname1"}
	assert.Equal(t, expectedInfo, info)

	_, ok := <-sh.removeCh
	expectedChannelCloseStatus := false
	assert.Equal(t, expectedChannelCloseStatus, ok, "channel is not closed")
}

func TestFileFilterErrCallbakc(t *testing.T) {
	testCases := []struct {
		alias    string
		fullpath string
		inputErr error
		want     ErrorAction
	}{
		{
			alias:    "custom error",
			fullpath: "/path",
			inputErr: errors.New("custom error"),
			want:     SkipNode,
		},
	}

	for _, tc := range testCases {
		sh, _ := NewShrinker(&Config{
			VerboseOutput: false,
		})
		t.Run(tc.alias, func(t *testing.T) {
			actionCode := sh.fileFilterErrCallback(tc.fullpath, tc.inputErr)
			assert.Equal(t, tc.want, actionCode, fmt.Sprintf("Input: %v", tc.inputErr))
		})
	}
}

func TestCleanerEmptyInput(t *testing.T) {
	assert := assert.New(t)
	sh, _ := NewShrinker(&Config{
		CheckPath:     "/here",
		VerboseOutput: false,
	})
	go sh.cleaner(func() {
		close(sh.statsCh)
	})

	close(sh.removeCh)
	stats, ok := <-sh.statsCh

	expectedChannelCloseStatus := false
	expectedFile := fs.FileStat{}

	assert.Equal(expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(&expectedFile, &stats, fmt.Sprintf("Input: %v", "empty"))
}

func TestCleanerBasicRemoveFile(t *testing.T) {
	assert := assert.New(t)

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock

	osMock.On("Stat", "/here/node_modules", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/test1", false).Return(fs.NewFileStat("test1", "/test1", 1, 1), nil)
	osMock.On("RemoveAll", "/test1").Return(nil)

	sh, _ := NewShrinker(&Config{
		CheckPath:     "/here",
		VerboseOutput: false,
	})
	go sh.cleaner(func() {
		close(sh.statsCh)
	})

	input := []removeObjInfo{
		{isDir: false, filename: "test1", fullpath: "/test1"},
	}

	for _, file := range input {
		sh.removeCh <- &file
	}

	close(sh.removeCh)

	stats, ok := <-sh.statsCh
	expectedChannelCloseStatus := true
	expectedFile := fs.NewFileStat("test1", "/test1", 1, 1)

	assert.Equal(expectedChannelCloseStatus, ok, "is channel open")
	assert.Equal(expectedFile, &stats, fmt.Sprintf("Input: %v", "empty"))

	// checking is channel properly closed
	expectedChannelCloseStatus2 := false
	_, ok2 := <-sh.statsCh
	assert.Equal(expectedChannelCloseStatus2, ok2, "is channel closed")

	osMock.AssertExpectations(t)
}

func TestCleanerBasicRemoveDirectory(t *testing.T) {
	assert := assert.New(t)

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock

	osMock.On("Stat", "/node_modules", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/dir1", true).Return(fs.NewFileStat("dir1", "/dir1", 1, 1), nil)
	osMock.On("RemoveAll", "/dir1").Return(nil)

	sh, _ := NewShrinker(&Config{
		CheckPath:     "/",
		VerboseOutput: false,
	})
	go sh.cleaner(func() {
		close(sh.statsCh)
	})

	input := []removeObjInfo{
		{isDir: true, filename: "dir1", fullpath: "/dir1"},
	}

	for _, file := range input {
		sh.removeCh <- &file
	}

	close(sh.removeCh)

	stats, ok := <-sh.statsCh
	expectedChannelCloseStatus := true
	expectedFile := fs.NewFileStat("dir1", "/dir1", 1, 1)

	assert.Equal(expectedChannelCloseStatus, ok, "is channel open")
	assert.Equal(expectedFile, &stats, fmt.Sprintf("Input: %v", "empty"))

	// checking is channel properly closed
	expectedChannelCloseStatus2 := false
	_, ok2 := <-sh.statsCh
	assert.Equal(expectedChannelCloseStatus2, ok2, "is channel closed")

	osMock.AssertExpectations(t)
}

func TestCleanerStatFileWithError(t *testing.T) {
	assert := assert.New(t)

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock

	osMock.On("Stat", "/node_modules", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/test2", false).Return(nil, errors.New("some error"))

	sh, _ := NewShrinker(&Config{
		CheckPath:     "/",
		VerboseOutput: false,
	})
	go sh.cleaner(func() {
		close(sh.statsCh)
	})

	input := []removeObjInfo{
		{isDir: false, filename: "test2", fullpath: "/test2"},
	}

	for _, file := range input {
		sh.removeCh <- &file
	}

	close(sh.removeCh)

	stats, ok := <-sh.statsCh
	expectedChannelCloseStatus := false
	expectedFile := &fs.FileStat{}

	assert.Equal(expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(expectedFile, &stats, fmt.Sprintf("Input: %v", "empty"))

	osMock.AssertExpectations(t)
}

func TestCleanerRemoveFileWithError(t *testing.T) {
	assert := assert.New(t)

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock

	osMock.On("Stat", "/node_modules", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/test3", false).Return(fs.NewFileStat("test3", "/test3", 1, 1), nil)
	osMock.On("RemoveAll", "/test3").Return(errors.New("custom error"))

	sh, _ := NewShrinker(&Config{
		CheckPath:     "/",
		VerboseOutput: false,
	})
	go sh.cleaner(func() {
		close(sh.statsCh)
	})

	input := []removeObjInfo{
		{isDir: false, filename: "test3", fullpath: "/test3"},
	}

	for _, file := range input {
		sh.removeCh <- &file
	}

	close(sh.removeCh)

	stats, ok := <-sh.statsCh
	expectedChannelCloseStatus := false
	expectedFile := &fs.FileStat{}

	assert.Equal(expectedChannelCloseStatus, ok, "is channel closed")
	assert.Equal(expectedFile, &stats, fmt.Sprintf("Input: %v", "empty"))

	osMock.AssertExpectations(t)
}

// TODO: write logic for separate logger testing
func TestLogger(t *testing.T) {

}

func TestStartFunc(t *testing.T) {
	testCases := []struct {
		alias    string
		waitResp bool
		want     *fs.FileStat
		wantErr  error
	}{
		{
			alias:    "check basic path existence",
			waitResp: false,
			want:     nil,
			wantErr:  errors.New("path doesn`t exist"),
		},
	}

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock
	// osMock.On("Getwd").Return("/here", nil)
	osMock.On("Stat", "/here/node_modules", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/here", false).Return(nil, os.ErrNotExist)

	for _, tc := range testCases {
		sh, _ := NewShrinker(&Config{
			CheckPath: "/here",
		})
		t.Run(tc.alias, func(t *testing.T) {
			err := sh.Start()
			if tc.waitResp {
				go func() {
					stats := <-sh.statsCh
					assert.Equal(t, tc.want, &stats)
				}()
			}
			assert.Equal(t, tc.wantErr, err)
		})
	}

	osMock.AssertExpectations(t)
}
