package shrunk

type Config struct {
	VerboseOutput  bool
	DryRun         bool
	ConcurentLimit int
	CheckPath      string
	RemoveFileExt  []string
	ExcludeNames   []string
	IncludeNames   []string
}
