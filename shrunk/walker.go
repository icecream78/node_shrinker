package shrunk

import (
	"github.com/karrick/godirwalk"
)

type Walker interface {
	Walk(string, godirwalk.WalkFunc, func(string, error) godirwalk.ErrorAction) error
}

type dirWalker struct {
}

func newDirWalker() *dirWalker {
	return &dirWalker{}
}

func (dw *dirWalker) Walk(filepath string, callback godirwalk.WalkFunc, errCallback func(string, error) godirwalk.ErrorAction) error {
	err := godirwalk.Walk(filepath, &godirwalk.Options{
		Unsorted:      true, // for higher speed walking dir tree
		Callback:      callback,
		ErrorCallback: errCallback,
	})
	return err
}
