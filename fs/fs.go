package fs

import (
	"os"

	. "github.com/icecream78/node_shrunker/walker"
)

type FS interface {
	Getwd() (string, error)
	Stat(filepath string, recursive bool) (*FileStat, error)
	RemoveAll(filepath string) error
	Remove(filepath string) error
}

func NewFS() *fsClass {
	return &fsClass{}
}

type fsClass struct {
}

func (o *fsClass) Stat(filepath string, recursive bool) (*FileStat, error) {
	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	return &FileStat{
		filename: stat.Name(),
		fullpath: filepath,
		size:     stat.Size(),
	}, nil
}

func (o *fsClass) getRecursiveStat(filepath string) (*FileStat, error) {
	stats := FileStat{filename: filepath, fullpath: filepath}
	err := NewDirWalker().Walk(filepath, func(path string, de FileInfoI) error {
		st, stErr := o.Stat(filepath, de.IsDir())
		if stErr != nil {
			// cannnot get stat from file, so we cannot remove it and not count this file in result stats
			return nil
		}

		stats.size += st.Size()
		stats.filesCount += st.filesCount
		return nil
	}, func(string, error) ErrorAction {
		return SkipNode
	})
	return &stats, err
}

func (o *fsClass) RemoveAll(filepath string) error {
	return os.RemoveAll(filepath)
}

func (o *fsClass) Remove(filepath string) error {
	return os.Remove(filepath)
}

func (o *fsClass) Getwd() (string, error) {
	return os.Getwd()
}
