package walker

import (
	"errors"

	"github.com/karrick/godirwalk"
)

type ErrorAction int

const (
	Halt ErrorAction = iota
	SkipNode
)

var (
	ExcludeError    = errors.New("file is in exclude list")
	SkipDirError    = errors.New("skipping recursive dir walk")
	NotProcessError = errors.New("not processing file")
)

type WalkFunc func(osPathname string, directoryEntry FileInfoI) error
type WalkErrFunc func(osPathname string, err error) ErrorAction

type Walker interface {
	Walk(path string, callback WalkFunc, errCallback WalkErrFunc) error
}

type dirWalker struct {
	keepOrder bool
}

func NewDirWalker(keepOrder bool) *dirWalker {
	return &dirWalker{keepOrder}
}

func (dw *dirWalker) Walk(filepath string, callback WalkFunc, errCallback WalkErrFunc) error {
	err := godirwalk.Walk(filepath, &godirwalk.Options{
		Unsorted: !dw.keepOrder, // for higher speed walking dir tree
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			err := callback(osPathname, NewFileInfoFromDe(de))

			// for library copability
			if err == NotProcessError {
				return nil
			}
			return err
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			answer := errCallback(osPathname, err)
			return godirwalk.ErrorAction(answer)
		},
	})
	return err
}
