package shrunk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathExists(t *testing.T) {
	osMock := new(MockOsI)

	osManager = osMock
	osMock.On("Stat", "/test1").Return(&FileStat{
		filename: "/test1",
	}, nil)
	osMock.On("Stat", "/test13").Return(nil, os.ErrNotExist)

	assert.Equal(t, pathExists("/test1"), true)
	assert.Equal(t, pathExists("/test13"), false)

	osMock.AssertExpectations(t)
}
