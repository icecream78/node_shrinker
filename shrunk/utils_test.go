package shrunk

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestPathExists(t *testing.T) {
	osMock := osManagerMock{}

	fs := map[string]*FileStat{}
	fs["/test1"] = &FileStat{
		filename: "/test1",
	}

	osMock.setFileStructure(fs)
	osMock.On("Stat", mock.AnythingOfType("string")).Return(osMock.getFileStats)
	osMock.On("Exit", mock.AnythingOfType("int")).Return()

	// osManager = osMock

	// assert.Equal(t, )
	osMock.AssertExpectations(t)
}
