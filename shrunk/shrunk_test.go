package shrunk

import (
	"os"
	"testing"

	"github.com/icecream78/node_shrunker/mocks"
)

type osManagerMock struct {
	mocks.OsI
	fileStructure map[string]*FileStat
}

func (o *osManagerMock) setFileStructure(structure map[string]*FileStat) {
	o.fileStructure = structure
}

func (o *osManagerMock) getFileStats(filepath string) (*FileStat, error) {
	file, exists := o.fileStructure[filepath]
	if !exists {
		return nil, os.ErrNotExist
	}

	return file, nil
}

func TestSimpleCase(t *testing.T) {
	// osMock := new(osManagerMock)
	// osMock.On("Getwd").Return("/test/dir", nil)
	// osMock.On("Stat", mock.AnythingOfType("string")).Return(osMock.getFileStats)
	// osMock.setFileStructure(map[string]*FileStat{})

	// err := NewShrunker(&Config{
	// 	CheckPath:       "some path",
	// 	RemoveDirNames:  []string{},
	// 	RemoveFileNames: []string{},
	// 	VerboseOutput:   false,
	// 	ExcludeNames:    []string{},
	// 	IncludeNames:    []string{},
	// }).Start()
	// if err != nil {
	// 	fmt.Printf("Someghing broken=) %v\n", err)
	// }
}
