package shrunk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/icecream78/node_shrunker/fs"
	"github.com/icecream78/node_shrunker/mocks"

	. "github.com/icecream78/node_shrunker/walker"
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
	sh := NewShrunker(&Config{
		ExcludeNames: []string{
			"file",
			"/a/b/c/file",
		},
	})
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
			assert.Equal(t, tc.want, sh.isExcludeName(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestRemoveDirNameFunc(t *testing.T) {
	sh := NewShrunker(&Config{
		RemoveDirNames: []string{
			"dirname",
			"/a/b/c/dirname",
		},
	})
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
			assert.Equal(t, tc.want, sh.isDirToRemove(tc.name), fmt.Sprintf("Input: %s", tc.name))
		})
	}
}

func TestRemoveFileNameFunc(t *testing.T) {
	sh := NewShrunker(&Config{
		RemoveFileNames: []string{
			"file",
			"/a/b/c/file",
		},
		RemoveFileExt: []string{
			".js",
			"js",
		},
	})
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
			assert.Equal(t, tc.want, sh.isFileToRemove(tc.name), fmt.Sprintf("Input: %s", tc.name))
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
		sh := NewShrunker(&Config{})
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

func TestFileFilterCallbakc(t *testing.T) {
	testCases := []struct {
		alias    string
		fullpath string
		input    testFileInfo
		want     *removeObjInfo
		waitResp bool
		err      error
	}{
		{
			alias:    "Check excluded file",
			fullpath: "/file1",
			input:    testFileInfo{name: "file1", isDir: false, isRegular: true},
			want:     nil,
			waitResp: false,
			err:      ExcludeError,
		},
		{
			alias:    "Check file remove",
			fullpath: "/file2",
			input:    testFileInfo{name: "file2", isDir: false, isRegular: true},
			want:     &removeObjInfo{isDir: false, filename: "file2", fullpath: "/file2"},
			waitResp: true,
			err:      ExcludeError,
		},
		{
			alias:    "Check dir not removed",
			fullpath: "/dirname",
			input:    testFileInfo{name: "dirname", isDir: true, isRegular: false},
			want:     &removeObjInfo{isDir: true, filename: "dirname", fullpath: "/dirname"},
			waitResp: false,
			err:      NotProcessError,
		},
		{
			alias:    "Check dirname remove",
			fullpath: "/dirname1",
			input:    testFileInfo{name: "dirname1", isDir: true, isRegular: false},
			want:     &removeObjInfo{isDir: true, filename: "dirname1", fullpath: "/dirname1"},
			waitResp: true,
			err:      SkipDirError,
		},
	}
	for _, tc := range testCases {
		sh := NewShrunker(&Config{
			ExcludeNames: []string{
				"file1",
			},
			RemoveDirNames: []string{
				"dirname1",
			},
			RemoveFileNames: []string{
				"file2",
			},
		})
		t.Run(tc.alias, func(t *testing.T) {
			if tc.waitResp {
				go func() {
					info := <-sh.removeCh
					assert.Equal(t, tc.want, info, fmt.Sprintf("Input: %v", tc.input))
				}()
			}

			err := sh.fileFilterCallback(tc.fullpath, testFileInfo{
				name:      tc.input.name,
				isDir:     tc.input.isDir,
				isRegular: tc.input.isRegular,
			})
			if err != nil {
				if tc.err != nil {
					// normal case
					return
				} else {
					t.Errorf("Not correct")
					return
				}
			}
		})
	}
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
		sh := NewShrunker(&Config{
			VerboseOutput: true,
		})
		t.Run(tc.alias, func(t *testing.T) {
			actionCode := sh.fileFilterErrCallback(tc.fullpath, tc.inputErr)
			assert.Equal(t, tc.want, actionCode, fmt.Sprintf("Input: %v", tc.inputErr))
		})
	}
}

func TestCleanerFunc(t *testing.T) {
	testCases := []struct {
		alias    string
		input    []removeObjInfo
		want     *fs.FileStat
		waitResp bool
	}{
		{
			alias:    "Check empty input files list",
			input:    []removeObjInfo{},
			waitResp: true,
			want:     fs.NewFileStat("result", "result", 0, 0),
		},
		{
			alias: "Check basic remove file",
			input: []removeObjInfo{
				{isDir: false, filename: "test1", fullpath: "/test1"},
			},
			waitResp: true,
			want:     fs.NewFileStat("test1", "/test1", 1, 1),
		},
		// TODO: fix test failing with goroutines
		// {
		// 	alias: "Check basic remove directory",
		// 	input: []removeObjInfo{
		// 		{isDir: true, filename: "dir1", fullpath: "/dir1"},
		// 	},
		// 	waitResp: true,
		// 	want:     fs.NewFileStat("dir1", "/dir1", 1, 1),
		// },
		{
			alias: "Check file with error",
			input: []removeObjInfo{
				{isDir: false, filename: "test2", fullpath: "/test2"},
			},
			waitResp: true,
			want:     nil,
		},
		{
			alias: "Check remove file with error",
			input: []removeObjInfo{
				{isDir: false, filename: "test3", fullpath: "/test3"},
			},
			waitResp: true,
			want:     nil,
		},
	}

	// prepare test
	osMock := new(mocks.FS)
	fsManager = osMock
	osMock.On("Stat", "/test1", false).Return(fs.NewFileStat("test1", "/test1", 1, 1), nil)
	osMock.On("Stat", "/test2", false).Return(fs.NewFileStat("", "", 0, 0), errors.New("some error"))
	// osMock.On("Stat", "/dir1", true).Return(fs.NewFileStat("dir1", "/dir1", 1, 1), nil)
	osMock.On("Stat", "/test3", false).Return(fs.NewFileStat("test3", "/test3", 1, 1), nil)
	osMock.On("Getwd").Return("/here", nil)
	osMock.On("RemoveAll", "/test1").Return(nil)
	// osMock.On("RemoveAll", "/dir1").Return(nil)
	osMock.On("RemoveAll", "/test3").Return(errors.New("custom error"))

	for _, tc := range testCases {
		sh := NewShrunker(&Config{
			VerboseOutput: true,
		})
		go sh.cleaner(func() {})
		t.Run(tc.alias, func(t *testing.T) {
			if tc.waitResp {
				go func() {
					stats := <-sh.statsCh
					close(sh.statsCh)
					assert.Equal(t, tc.want, &stats, fmt.Sprintf("Input: %v", tc.input))
				}()
			}
			for _, file := range tc.input {
				sh.removeCh <- &file
			}
		})
	}
	osMock.AssertExpectations(t)
}
