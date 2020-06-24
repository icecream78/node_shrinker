package shrunk

import "os"

type FileStat struct {
	filename string
	fullpath string
	size     int64
}

func (fs *FileStat) Size() int64 {
	return fs.size
}

type OsI interface {
	Getwd() (string, error)
	Stat(filepath string) (*FileStat, error)
	RemoveAll(filepath string) error
	Remove(filepath string) error
	Exit(exitCode int)
}

func newOs() *osClass {
	return &osClass{}
}

type osClass struct {
}

func (o *osClass) Stat(filepath string) (*FileStat, error) {
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

func (o *osClass) RemoveAll(filepath string) error {
	return os.RemoveAll(filepath)
}

func (o *osClass) Remove(filepath string) error {
	return os.Remove(filepath)
}

func (o *osClass) Exit(exitCode int) {
	os.Exit(exitCode)
}

func (o *osClass) Getwd() (string, error) {
	return os.Getwd()
}
