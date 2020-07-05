package fs

type SizeFormat int64

const (
	BytesFormat     SizeFormat = 1
	KilobytesFormat SizeFormat = 1024
	MegabyesFormat  SizeFormat = 1024 * 1024
)

type FileStat struct {
	filename   string
	fullpath   string
	size       int64
	filesCount int64
}

func (fs *FileStat) Size() int64 {
	return fs.size
}

func NewFileStat(filename, fullpath string, size int64, filesCount int64) *FileStat {
	return &FileStat{
		filename:   filename,
		fullpath:   fullpath,
		size:       size,
		filesCount: filesCount,
	}
}

func (fs *FileStat) GetHumanSizeFormat(format SizeFormat) int64 {
	return fs.size / int64(format)
}

func (fs *FileStat) FilesCount() int64 {
	return fs.filesCount
}
