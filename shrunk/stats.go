package shrunk

import (
	"github.com/karrick/godirwalk"
)

type sizeFormat int64

const (
	bytesFormat     sizeFormat = 1
	kilobytesFormat sizeFormat = 1024
	megabyesFormat  sizeFormat = 1024 * 1024
)

type dirStats struct {
	size         int64
	removedCount int
}

func (s *dirStats) getHumanSizeFormat(format sizeFormat) int64 {
	return s.size / int64(format)
}

func getDirectoryStats(filepath string) (*dirStats, error) {
	stats := dirStats{}
	err := newDirWalker().Walk(filepath, func(path string, de *godirwalk.Dirent) error {
		if de.IsDir() {
			return nil
		}
		st, stErr := osManager.Stat(filepath)
		if stErr != nil {
			// cannnot get stat from file, so we cannot remove it and not count this file in result stats
			return nil
		}

		stats.size = st.Size()
		stats.removedCount++
		return nil
	}, func(string, error) godirwalk.ErrorAction {
		return godirwalk.SkipNode
	})
	return &stats, err
}

func getFileStats(filepath string) (*dirStats, error) {
	st, _ := osManager.Stat(filepath)
	return &dirStats{
		size:         st.Size(),
		removedCount: 1,
	}, nil
}
