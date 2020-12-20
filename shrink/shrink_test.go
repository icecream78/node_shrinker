package shrink

import (
	"context"
	"fmt"
	"testing"

	"github.com/icecream78/node_shrinker/fs"

	. "github.com/icecream78/node_shrinker/walker"
	"github.com/stretchr/testify/assert"
)

type loggerStub struct {
}

func newLoggerStub() *loggerStub {
	return &loggerStub{}
}

func (l *loggerStub) Infof(format string, a ...interface{}) {}
func (l *loggerStub) Infoln(a ...interface{})               {}

type walkerStub struct {
	testPathes map[string]string
}

func (w *walkerStub) SetFileStructure(pathes map[string]string) {
	w.testPathes = pathes
}

func (w *walkerStub) Walk(filepath string, callback WalkFunc, errCallback WalkErrFunc) error {
	for path := range w.testPathes {
		_ = callback(path, &FileInfo{})
	}
	return nil
}

func TestStatGrabberFunc(t *testing.T) {
	testCases := []struct {
		alias string
		input []*fs.FileStat
		want  *fs.FileStat
	}{
		{"Check stats grabber with empty input", []*fs.FileStat{}, fs.NewFileStat("result", "result", 0, 0)},
		{"Put few files for calculations", []*fs.FileStat{
			fs.NewFileStat("1", "1", 1024, 1),
			fs.NewFileStat("2", "2", 1024, 1),
		}, fs.NewFileStat("result", "result", 2048, 2)},
	}
	for _, tc := range testCases {
		sh, err := NewShrinker(&Config{ConcurentLimit: 1, CheckPath: "/"}, newLoggerStub())
		if err != nil {
			t.Errorf("Got error: %v", err)
			continue
		}

		ctx := context.TODO()

		t.Run(tc.alias, func(t *testing.T) {
			processCh := make(chan *fs.FileStat)
			resStatsCh := sh.runStatGrabber(ctx, processCh)

			for _, file := range tc.input {
				processCh <- file
			}

			close(processCh)

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
