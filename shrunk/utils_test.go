package shrunk

import (
	"fmt"
	"os"
	"testing"

	"github.com/icecream78/node_shrinker/fs"

	"github.com/icecream78/node_shrinker/mocks"

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

func TestSplitPatternsFunc(t *testing.T) {
	testCases := []struct {
		input            []string
		expectedPatterns []string
		expectedRegular  []string
	}{
		{
			input:            []string{"script", "script1.js"},
			expectedRegular:  []string{"script", "script1.js"},
			expectedPatterns: []string{},
		},
		{
			input:            []string{"script", "script1.js", "*scr*", "/tmp/a?c"},
			expectedRegular:  []string{"script", "script1.js"},
			expectedPatterns: []string{"*scr*", "/tmp/a?c"},
		},
		{
			input:            []string{"*scr*", "/tmp/a?c"},
			expectedRegular:  []string{},
			expectedPatterns: []string{"*scr*", "/tmp/a?c"},
		},
	}
	for _, tc := range testCases {
		patterns, regular := devidePatternsFromRegularNames(tc.input)

		assert.Equal(t, patterns, tc.expectedPatterns, fmt.Sprintf("Input: %v", tc.input))
		assert.Equal(t, regular, tc.expectedRegular, fmt.Sprintf("Input: %v", tc.input))
	}
}
