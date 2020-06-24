package shrunk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathExists(t *testing.T) {
	osMock := newOsManagerMock()

	fs := map[string]*FileStat{}
	fs["/test1"] = &FileStat{
		filename: "/test1",
	}

	osMock.setFileStructure(fs)
	// osMock.On("Stat", mock.AnythingOfType("string")).Return(osMock.getFileStats)
	osMock.On("Stat", "/test1").Return(FileStat{
		filename: "/test1",
	}, nil)

	osManager = osMock

	assert.Equal(t, pathExists("/test1"), true)
	assert.Equal(t, pathExists("/test13"), false)

	osMock.AssertExpectations(t)
}
