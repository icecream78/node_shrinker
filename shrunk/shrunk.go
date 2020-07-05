package shrunk

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	. "github.com/icecream78/node_shrunker/fs"
	. "github.com/icecream78/node_shrunker/walker"
)

var fsManager FS = NewFS() // for test purposes
var walker Walker = NewDirWalker()

type removeObjInfo struct {
	isDir    bool
	filename string
	fullpath string
}

type Shrunker struct {
	verboseOutput   bool
	concurentLimit  int
	checkPath       string
	shrunkDirNames  map[string]struct{}
	shrunkFileNames map[string]struct{}
	shrunkFileExt   map[string]struct{}
	excludeNames    map[string]struct{}
	removeCh        chan *removeObjInfo
	statsCh         chan FileStat
}

func NewShrunker(cfg *Config) *Shrunker {
	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 4
	}
	checkPath := cfg.CheckPath
	if checkPath == "" {
		path, _ := fsManager.Getwd()
		checkPath = filepath.Join(path, "node_modules")
	}

	return &Shrunker{
		verboseOutput:   cfg.VerboseOutput,
		checkPath:       checkPath,
		shrunkDirNames:  sliceToMap(DefaultRemoveDirNames, cfg.RemoveDirNames, cfg.IncludeNames),
		shrunkFileNames: sliceToMap(DefaultRemoveFileNames, cfg.RemoveFileNames, cfg.IncludeNames),
		shrunkFileExt:   sliceToMap(DefaultRemoveFileExt, cfg.RemoveFileExt),
		excludeNames:    sliceToMap(cfg.ExcludeNames),
		removeCh:        make(chan *removeObjInfo),
		statsCh:         make(chan FileStat),
		concurentLimit:  concurentLimit,
	}
}

func (sh *Shrunker) isExcludeName(name string) bool {
	_, exists := sh.excludeNames[name]
	return exists
}

func (sh *Shrunker) isDirToRemove(name string) bool {
	_, exists := sh.shrunkDirNames[name]
	return exists
}

func (sh *Shrunker) isFileToRemove(name string) (exists bool) {
	if _, exists = sh.shrunkFileNames[name]; exists {
		return
	}
	ext := filepath.Ext(name)
	if ext == name { // for cases, when files starts with leading dot
		return
	}
	if _, exists = sh.shrunkFileExt[ext]; exists {
		return
	}
	return
}

func (sh *Shrunker) cleaner(done func()) {
	var err error
	var obj *removeObjInfo
	var stat *FileStat

	for obj = range sh.removeCh {
		if sh.verboseOutput {
			fmt.Printf("removing: %s\n", obj.fullpath)
		}

		if obj.isDir {
			stat, err = fsManager.Stat(obj.fullpath, true)
		} else {
			stat, err = fsManager.Stat(obj.fullpath, false)
		}

		if err != nil {
			if sh.verboseOutput {
				fmt.Printf("ERROR: %s\n", err)
			}
			continue
		}

		if err = fsManager.RemoveAll(obj.fullpath); err != nil {
			if sh.verboseOutput {
				fmt.Printf("ERROR: %s\n", err)
			}
			continue
		}
		sh.statsCh <- *stat
	}
	done()
}

func (sh *Shrunker) runCleaners() (err error) {
	var wg sync.WaitGroup
	wg.Add(sh.concurentLimit)
	for i := 0; i < sh.concurentLimit; i++ {
		go sh.cleaner(wg.Done)
	}
	wg.Wait()
	close(sh.statsCh)
	return nil
}

func (sh *Shrunker) runStatGrabber() chan FileStat {
	resCh := make(chan FileStat)

	go func(resCh chan FileStat) {
		var stat FileStat
		var removedCount int64
		var removedSize int64
		for stat = range sh.statsCh {
			removedCount += stat.FilesCount()
			removedSize += stat.Size()
		}
		resCh <- *NewFileStat("result", "result", removedSize, removedCount)
	}(resCh)

	return resCh
}

func (sh *Shrunker) Start() error {
	return sh.start()
}

// TODO: add errors wrapping for correct handling errors
func (sh *Shrunker) fileFilterCallback(osPathname string, de FileInfoI) error {
	if sh.isExcludeName(de.Name()) {
		return ExcludeError
	}

	if de.IsDir() && sh.isDirToRemove(de.Name()) {
		sh.removeCh <- &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}
		return SkipDirError
	} else if de.IsRegular() && sh.isFileToRemove(de.Name()) {
		sh.removeCh <- &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}
	}
	return NotProcessError
}

func (sh *Shrunker) fileFilterErrCallback(osPathname string, err error) ErrorAction {
	// TODO: more informative logging about errors
	if sh.verboseOutput {
		fmt.Printf("ERROR: %s\n", err)
	}
	return SkipNode
}

// TODO: think how add progress bar
func (sh *Shrunker) start() error {
	if !pathExists(sh.checkPath) {
		fmt.Printf("Path %s doesn`t exist\n", sh.checkPath)
		return nil
	}

	go sh.runCleaners()
	statsCh := sh.runStatGrabber()

	fmt.Printf("Start checking directory %s\n", sh.checkPath)
	err := walker.Walk(sh.checkPath, sh.fileFilterCallback, sh.fileFilterErrCallback)

	close(sh.removeCh)

	stats := <-statsCh

	if err != nil {
		// TODO: write error handler with case checking
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	fmt.Println("Remove stats:")
	fmt.Printf("total removed: %d MB\n", stats.GetHumanSizeFormat(MegabyesFormat))
	fmt.Printf("files removed: %d\n", stats.FilesCount())
	return err
}
