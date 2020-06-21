package shrunk

import "os"

type osI interface {
	Getwd() (string, error)
	Stat(filepath string) (os.FileInfo, error)
	RemoveAll(filepath string) error
	Remove(filepath string) error
	Exit(exitCode int)
}

func newOs() *osClass {
	return &osClass{}
}

type osClass struct {
}

func (o *osClass) Stat(filepath string) (os.FileInfo, error) {
	return os.Stat(filepath)
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
