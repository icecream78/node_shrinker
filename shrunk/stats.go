package shrunk

import (
	"os"

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
	err := godirwalk.Walk(filepath, &godirwalk.Options{
		Unsorted: true, // for higher speed walking dir tree
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}
			st, _ := os.Stat(filepath)
			stats.size = st.Size()
			stats.removedCount++
			return nil
		},
	})
	return &stats, err
}

func getFileStats(filepath string) (*dirStats, error) {
	st, _ := os.Stat(filepath)
	return &dirStats{
		size:         st.Size(),
		removedCount: 1,
	}, nil
}
