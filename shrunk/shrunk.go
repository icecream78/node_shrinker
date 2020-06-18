package shrunk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/karrick/godirwalk"
)

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
	removeCh        chan *removeObjInfo
	statsCh         chan dirStats
}

func NewShrunker(cfg *Config) *Shrunker {
	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 4
	}
	checkPath := cfg.CheckPath
	if checkPath == "" {
		path, _ := os.Getwd()
		checkPath = filepath.Join(path, "node_modules")
	}

	return &Shrunker{
		verboseOutput:   cfg.VerboseOutput,
		checkPath:       checkPath,
		shrunkDirNames:  sliceToMap(DefaultRemoveDirNames, cfg.RemoveDirNames),
		shrunkFileNames: sliceToMap(DefaultRemoveFileNames, cfg.RemoveFileNames),
		removeCh:        make(chan *removeObjInfo),
		statsCh:         make(chan dirStats),
		concurentLimit:  concurentLimit,
	}
}

func (sh *Shrunker) runCleaners() error {
	var wg sync.WaitGroup
	wg.Add(sh.concurentLimit)
	for i := 0; i < sh.concurentLimit; i++ {
		go func(done func()) {
			var obj *removeObjInfo
			var err error
			for obj = range sh.removeCh {
				if sh.verboseOutput {
					fmt.Printf("removing: %s\n", obj.fullpath)
				}

				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
					continue
				}
				if obj.isDir {
					stat, _ := getDirectoryStats(obj.fullpath)
					err = os.RemoveAll(obj.fullpath)
					sh.statsCh <- *stat
				} else {
					stat, _ := getFileStats(obj.fullpath)
					err = os.Remove(obj.fullpath)
					sh.statsCh <- *stat
				}
			}
			done()
		}(wg.Done)
	}
	wg.Wait()
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
	if de.IsDir() {
		if _, exists := sh.shrunkDirNames[de.Name()]; exists {
			sh.removeCh <- &removeObjInfo{
				isDir:    de.IsDir(),
				filename: de.Name(),
				fullpath: osPathname,
			}
			return errors.New("skip dir " + osPathname)
		}
	} else if de.IsRegular() {
		if _, exists := sh.shrunkFileNames[de.Name()]; exists {
			sh.removeCh <- &removeObjInfo{
				isDir:    de.IsDir(),
				filename: de.Name(),
				fullpath: osPathname,
			}
		}
	}
	return nil
}

func (sh *Shrunker) fileFilterErrCallback(osPathname string, err error) godirwalk.ErrorAction {
	// TODO: more informative logging about errors
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
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
	err := godirwalk.Walk(sh.checkPath, &godirwalk.Options{
		Unsorted:      true, // for higher speed walking dir tree
		Callback:      sh.fileFilterCallback,
		ErrorCallback: sh.fileFilterErrCallback,
	})

	close(sh.removeCh)
	close(sh.statsCh)

	stats := <-statsCh

	if err != nil {
		// TODO: write error handler with case checking
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	fmt.Println("Remove stats:")
	fmt.Printf("total removed: %d MB\n", stats.getHumanSizeFormat(megabyesFormat))
	fmt.Printf("files removed: %d\n", stats.removedCount)
	return err
}
