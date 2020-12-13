package shrunk

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"regexp"
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
	verboseOutput      bool
	dryRun             bool
	concurentLimit     int
	checkPath          string
	shrunkDirNames     map[string]struct{}
	shrunkFileNames    map[string]struct{}
	shrunkFileExt      map[string]struct{}
	excludeNames       map[string]struct{}
	regExpIncludeNames []*regexp.Regexp
	regExpExcludeNames []*regexp.Regexp
	removeCh           chan *removeObjInfo
	statsCh            chan FileStat
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

	patternInclude, regularInclude := devidePatternsFromRegularNames(cfg.IncludeNames)
	patternExclude, regularExclude := devidePatternsFromRegularNames(cfg.ExcludeNames)

	walker = NewDirWalker(cfg.DryRun)

	compiledIncludeRegList, err := compileRegExpList(patternInclude)
	if err != nil {
		return nil, err
	}

	compiledExcludeRegList, err := compileRegExpList(patternExclude)
	if err != nil {
		return nil, err
	}

	return &Shrinker{
		verboseOutput:      cfg.VerboseOutput,
		dryRun:             cfg.DryRun,
		checkPath:          checkPath,
		shrunkDirNames:     sliceToMap(DefaultRemoveDirNames, regularInclude),
		shrunkFileNames:    sliceToMap(DefaultRemoveFileNames, regularInclude),
		shrunkFileExt:      sliceToMap(DefaultRemoveFileExt, cfg.RemoveFileExt),
		excludeNames:       sliceToMap(regularExclude),
		regExpIncludeNames: compiledIncludeRegList,
		regExpExcludeNames: compiledExcludeRegList,
		removeCh:           make(chan *removeObjInfo, concurentLimit),
		statsCh:            make(chan FileStat, concurentLimit),
		concurentLimit:     concurentLimit,
	}, nil
}

func (sh *Shrinker) isExcludeName(name string) bool {
	_, exists := sh.excludeNames[name]
	if exists {
		return true
	}

	for _, pattern := range sh.regExpExcludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}

	return false
}

func (sh *Shrinker) isIncludeName(name string) bool {
	// return false
	for _, pattern := range sh.regExpIncludeNames {
		matched := pattern.MatchString(name)
		if matched {
			return true
		}
	}
	return false
}

func (sh *Shrinker) isDirToRemove(name string) bool {
	_, exists := sh.shrunkDirNames[name]
	return exists
}

func (sh *Shrinker) isFileToRemove(name string) (exists bool) {
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

func (sh *Shrinker) checkIsFileToProcess(de FileInfoI) (bool, error) {
	if sh.isExcludeName(de.Name()) {
		return false, ExcludeError
	}

	if sh.isIncludeName(de.Name()) {
		return true, nil
	} else if de.IsDir() && sh.isDirToRemove(de.Name()) {
		return true, SkipDirError
	} else if de.IsRegular() && sh.isFileToRemove(de.Name()) {
		return true, nil
	}
	return false, NotProcessError
}

// TODO: add errors wrapping for correct handling errors
func (sh *Shrinker) fileFilterCallback(osPathname string, de FileInfoI) error {
	isProcess, err := sh.checkIsFileToProcess(de)
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
		isProcess, _ := sh.checkIsFileToProcess(NewFileInfoFromOsFile(file))
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
