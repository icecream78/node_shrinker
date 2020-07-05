package walker

import "github.com/karrick/godirwalk"

type FileInfo struct {
	name      string
	isDir     bool
	isRegular bool
}

type FileInfoI interface {
	Name() string
	IsDir() bool
	IsRegular() bool
}

func (w *FileInfo) Name() string {
	return w.name
}

func (w *FileInfo) IsDir() bool {
	return w.isDir
}

func (w *FileInfo) IsRegular() bool {
	return w.isRegular
}

func NewFileInfoFromDe(de *godirwalk.Dirent) *FileInfo {
	return &FileInfo{
		name:      de.Name(),
		isDir:     de.IsDir(),
		isRegular: de.IsRegular(),
	}
}
