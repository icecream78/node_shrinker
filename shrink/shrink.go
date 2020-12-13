package shrink

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"sync"

	. "github.com/icecream78/node_shrinker/fs"
	. "github.com/icecream78/node_shrinker/walker"

	humanize "github.com/dustin/go-humanize"
	color "github.com/logrusorgru/aurora"
)

const NodeModulesDirname = "node_modules"
const (
	progressChar = "├───"
	lastChar     = "└───"
	tabChar      = "	"
)

var fsManager FS = NewFS() // for test purposes
var walker Walker

type removeObjInfo struct {
	isDir    bool
	filename string
	fullpath string
}

type Shrinker struct {
	verboseOutput  bool
	dryRun         bool
	concurentLimit int
	checkPath      string
	filter         *Filter
	removeCh       chan *removeObjInfo
	statsCh        chan FileStat
}

func NewShrinker(cfg *Config) (*Shrinker, error) {
	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 1
	}
	var checkPath string

	if cfg.CheckPath == "" {
		path, _ := fsManager.Getwd()
		checkPath = filepath.Join(path, NodeModulesDirname)
	} else if path.Base(cfg.CheckPath) == NodeModulesDirname {
		checkPath = cfg.CheckPath
	} else {
		if pathExists(filepath.Join(cfg.CheckPath, NodeModulesDirname)) {
			checkPath = filepath.Join(cfg.CheckPath, NodeModulesDirname)
		} else {
			checkPath = cfg.CheckPath
		}
	}

	walker = NewDirWalker(cfg.DryRun)

	return &Shrinker{
		verboseOutput:  cfg.VerboseOutput,
		dryRun:         cfg.DryRun,
		checkPath:      checkPath,
		filter:         NewFilter(cfg.IncludeNames, cfg.ExcludeNames, cfg.RemoveFileExt),
		removeCh:       make(chan *removeObjInfo, concurentLimit),
		statsCh:        make(chan FileStat, concurentLimit),
		concurentLimit: concurentLimit,
	}, nil
}

func (sh *Shrinker) cleaner(done func()) {
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

func (sh *Shrinker) runCleaners() (err error) {
	var wg sync.WaitGroup
	wg.Add(sh.concurentLimit)
	for i := 0; i < sh.concurentLimit; i++ {
		go sh.cleaner(wg.Done)
	}
	wg.Wait()
	close(sh.statsCh)
	return nil
}

func (sh *Shrinker) runStatGrabber() chan FileStat {
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

func (sh *Shrinker) Start() error {
	if !pathExists(sh.checkPath) {
		if sh.verboseOutput {
			fmt.Printf("Path %s doesn`t exist\n", sh.checkPath)
		}
		return errors.New("path doesn`t exist")
	}

	if sh.dryRun {
		return sh.startPrinter()
	}

	return sh.startCleaner()
}

// TODO: add errors wrapping for correct handling errors
func (sh *Shrinker) fileFilterCallback(osPathname string, de FileInfoI) error {
	isProcess, err := sh.filter.Check(de)
	if isProcess {
		ff := &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}
		sh.removeCh <- ff
	}
	if err != nil {
		return err
	}
	return nil
}

func (sh *Shrinker) fileFilterErrCallback(osPathname string, err error) ErrorAction {
	// TODO: more informative logging about errors
	if err == SkipDirError {
		return SkipNode
	}

	if sh.verboseOutput {
		fmt.Printf("ERROR: %s\n", err)
	}
	return SkipNode
}

func (sh *Shrinker) layoutPrinter(checkPath string, tabPassed string) error {
	files, err := ioutil.ReadDir(checkPath)
	if err != nil {
		log.Println(err)
	}

	filteredFiles := make([]string, 0)
	for _, file := range files {
		isProcess, _ := sh.filter.Check(NewFileInfoFromOsFile(file))
		if isProcess {
			filteredFiles = append(filteredFiles, file.Name())
		}
	}
	processedFiles := sliceToMap(filteredFiles)

	var tabToAdd, tabToPass, logLine string
	var printName, printFileSize interface{}
	var fileSize int64 = 0
	var fileStat FileStat

	for i, file := range files {
		printName = file.Name()

		if i == len(files)-1 {
			tabToAdd = lastChar
			tabToPass = " " + tabChar
		} else {
			tabToAdd = progressChar
			tabToPass = "│" + tabChar
		}

		tabToPass = tabPassed + tabToPass

		_, isFileInProcess := processedFiles[file.Name()]
		if file.IsDir() {
			if isFileInProcess {
				printName = color.Green(printName)
			} else {
				printName = color.Yellow(printName)
			}
			stat, err := fsManager.Stat(path.Join(checkPath, file.Name()), true)
			if err == nil {
				fileSize = stat.Size()
				fileStat = *stat
			} else {
				fileSize = 0
				fileStat = *NewFileStat(file.Name(), path.Join(checkPath, file.Name()), 0, 1)
			}
		}

		if !file.IsDir() {
			if isFileInProcess {
				printName = color.Green(printName)
			} else {
				printName = color.Red(printName)
			}
			fileSize = file.Size()
			fileStat = *NewFileStat(file.Name(), path.Join(checkPath, file.Name()), fileSize, 1)
		}

		if fileSize != 0 {
			printFileSize = color.Cyan(fmt.Sprintf("%v", humanize.Bytes(uint64(fileSize))))
		} else {
			printFileSize = color.Yellow("empty")
		}

		logLine = fmt.Sprintf("%v%v%v (%v)\n", tabPassed, tabToAdd, printName, printFileSize)
		fmt.Println(logLine)

		if isFileInProcess {
			sh.statsCh <- fileStat
		}

		// skip directories that matched by name
		if file.IsDir() && !isFileInProcess {
			nextDirPath := fmt.Sprintf("%v/%v", checkPath, file.Name())
			sh.layoutPrinter(nextDirPath, tabToPass)
		}
	}

	return nil
}

func (sh *Shrinker) startPrinter() (err error) {
	fmt.Printf("Start checking directory %s\n", color.Green(sh.checkPath))
	statsCh := sh.runStatGrabber()

	if err = sh.layoutPrinter(sh.checkPath, ""); err != nil {
		return
	}

	close(sh.statsCh)
	stats := <-statsCh
	close(statsCh)

	fmt.Println("Dry-run stats:")
	fmt.Printf("space to release: %v\n", color.Cyan(humanize.Bytes(uint64(stats.Size()))))
	fmt.Printf("files count to remove: %d\n", color.Cyan(stats.FilesCount()))
	return
}

func (sh *Shrinker) startCleaner() error {
	go sh.runCleaners()

	statsCh := sh.runStatGrabber()

	fmt.Printf("Start checking directory %s\n", color.Green(sh.checkPath))
	err := walker.Walk(sh.checkPath, sh.fileFilterCallback, sh.fileFilterErrCallback)

	close(sh.removeCh)

	stats := <-statsCh

	if err != nil {
		return err
	}

	fmt.Println("Remove stats:")
	fmt.Printf("released space: %v\n", color.Cyan(humanize.Bytes(uint64(stats.Size()))))
	fmt.Printf("files count: %d\n", color.Cyan(stats.FilesCount()))
	return err
}
