package shrunk

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/karrick/godirwalk"
)

var osManager osI = newOs() // for test purposes

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
	statsCh         chan dirStats

	walker Walker
}

func NewShrunker(cfg *Config) *Shrunker {
	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 4
	}
	checkPath := cfg.CheckPath
	if checkPath == "" {
		path, _ := osManager.Getwd()
		checkPath = filepath.Join(path, "node_modules")
	}

	return &Shrunker{
		verboseOutput:   cfg.VerboseOutput,
		checkPath:       checkPath,
		shrunkDirNames:  sliceToMap(DefaultRemoveDirNames, cfg.RemoveDirNames, cfg.IncludeNames),
		shrunkFileNames: sliceToMap(DefaultRemoveFileNames, cfg.RemoveFileNames, cfg.IncludeNames),
		shrunkFileExt:   sliceToMap(DefaultRemoveFileExt),
		excludeNames:    sliceToMap(cfg.ExcludeNames),
		removeCh:        make(chan *removeObjInfo),
		statsCh:         make(chan dirStats),
		concurentLimit:  concurentLimit,
		walker:          newDirWalker(),
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

func (sh *Shrunker) isFileToRemove(name string) bool {
	var exists bool
	if _, exists = sh.shrunkFileNames[name]; exists {
		return exists
	}
	ext := filepath.Ext(name)
	if _, exists = sh.shrunkFileExt[ext]; exists {
		return exists
	}
	return exists
}

func (sh *Shrunker) runCleaners() (err error) {
	var wg sync.WaitGroup
	wg.Add(sh.concurentLimit)
	for i := 0; i < sh.concurentLimit; i++ {
		go func(done func()) {
			var obj *removeObjInfo
			var stat *dirStats
			for obj = range sh.removeCh {
				if sh.verboseOutput {
					fmt.Printf("removing: %s\n", obj.fullpath)
				}

				if err != nil {
					if sh.verboseOutput {
						fmt.Printf("ERROR: %s\n", err)
					}
					continue
				}

				if obj.isDir {
					stat, _ = getDirectoryStats(obj.fullpath)
				} else {
					stat, _ = getFileStats(obj.fullpath)
				}
				if err = osManager.RemoveAll(obj.fullpath); err != nil {
					if sh.verboseOutput {
						fmt.Printf("ERROR: %s\n", err)
					}
					continue
				}
				sh.statsCh <- *stat
			}
			done()
		}(wg.Done)
	}
	wg.Wait()
	close(sh.statsCh)
	return nil
}

func (sh *Shrunker) runStatGrabber() chan dirStats {
	resCh := make(chan dirStats)

	go func(resCh chan dirStats) {
		var totalStats dirStats
		var stat dirStats
		for stat = range sh.statsCh {
			totalStats.removedCount += stat.removedCount
			totalStats.size += stat.size
		}
		resCh <- totalStats
	}(resCh)

	return resCh
}

func (sh *Shrunker) Start() error {
	return sh.start()
}

func (sh *Shrunker) fileFilterCallback(osPathname string, de *godirwalk.Dirent) error {
	if sh.isExcludeName(de.Name()) {
		return fmt.Errorf("file %s in exlclude list", osPathname)
	}

	if de.IsDir() && sh.isDirToRemove(de.Name()) {
		sh.removeCh <- &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}
		return errors.New("skip dir " + osPathname)
	} else if de.IsRegular() && sh.isFileToRemove(de.Name()) {
		sh.removeCh <- &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}
	}
	return nil
}

func (sh *Shrunker) fileFilterErrCallback(osPathname string, err error) godirwalk.ErrorAction {
	// TODO: more informative logging about errors
	if sh.verboseOutput {
		fmt.Printf("ERROR: %s\n", err)
	}
	return godirwalk.SkipNode
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
	err := sh.walker.Walk(sh.checkPath, sh.fileFilterCallback, sh.fileFilterErrCallback)

	close(sh.removeCh)

	stats := <-statsCh

	if err != nil {
		// TODO: write error handler with case checking
		fmt.Printf("%s\n", err)
		osManager.Exit(1)
	}

	fmt.Println("Remove stats:")
	fmt.Printf("total removed: %d MB\n", stats.getHumanSizeFormat(megabyesFormat))
	fmt.Printf("files removed: %d\n", stats.removedCount)
	return err
}
