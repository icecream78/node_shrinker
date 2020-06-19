package shrunk

type Config struct {
	VerboseOutput   bool
	ConcurentLimit  int
	CheckPath       string
	RemoveDirNames  []string
	RemoveFileNames []string
	RemoveFileExt   []string
}
