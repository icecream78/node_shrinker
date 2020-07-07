package shrunk

import (
	"os"
	"testing"

	"github.com/icecream78/node_shrunker/fs"

	"github.com/icecream78/node_shrunker/mocks"

	"github.com/stretchr/testify/assert"
)

func TestPathExists(t *testing.T) {
	osMock := new(mocks.FS)

	fsManager = osMock
	osMock.On("Stat", "/test1", false).Return(fs.NewFileStat("/test1", "/test1", 1, 1), nil)
	osMock.On("Stat", "/test13", false).Return(nil, os.ErrNotExist)
	osMock.On("Stat", "/test14", false).Return(nil, os.ErrPermission)

	assert.Equal(t, pathExists("/test1"), true)
	assert.Equal(t, pathExists("/test14"), true)
	assert.Equal(t, pathExists("/test13"), false)

	osMock.AssertExpectations(t)
}
