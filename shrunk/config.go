package shrunk

type Config struct {
	VerboseOutput  bool
	ConcurentLimit int
	CheckPath      string
	RemoveFileExt  []string
	ExcludeNames   []string
	IncludeNames   []string
}
