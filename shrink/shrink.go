package shrunk

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	. "github.com/icecream78/node_shrinker/fs"
	. "github.com/icecream78/node_shrinker/walker"
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

func NewShrinker(cfg *Config) *Shrinker {
	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 1
	}
	var checkPath string

	if cfg.CheckPath == "" {
		path, _ := fsManager.Getwd()
		checkPath = filepath.Join(path, NodeModulesDirname)
	} else if path.Base(cfg.CheckPath) != NodeModulesDirname {
		if pathExists(filepath.Join(checkPath, NodeModulesDirname)) {
			checkPath = filepath.Join(cfg.CheckPath, NodeModulesDirname)
		} else {
			checkPath = cfg.CheckPath
		}
	}

	patternInclude, regularInclude := devidePatternsFromRegularNames(cfg.IncludeNames)
	patternExclude, regularExclude := devidePatternsFromRegularNames(cfg.ExcludeNames)

	walker = NewDirWalker(cfg.DryRun)

	return &Shrinker{
		verboseOutput:      cfg.VerboseOutput,
		dryRun:             cfg.DryRun,
		checkPath:          checkPath,
		shrunkDirNames:     sliceToMap(DefaultRemoveDirNames, regularInclude),
		shrunkFileNames:    sliceToMap(DefaultRemoveFileNames, regularInclude),
		shrunkFileExt:      sliceToMap(DefaultRemoveFileExt, cfg.RemoveFileExt),
		excludeNames:       sliceToMap(regularExclude),
		regExpIncludeNames: compileRegExpList(patternInclude),
		regExpExcludeNames: compileRegExpList(patternExclude),
		removeCh:           make(chan *removeObjInfo, concurentLimit),
		statsCh:            make(chan FileStat, concurentLimit),
		concurentLimit:     concurentLimit,
	}
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

func (sh *Shrinker) printer(done func()) {
	var err error
	var obj *removeObjInfo
	var stat *FileStat

	for obj = range sh.removeCh {
		fmt.Printf("not removing: %s\n", obj.fullpath)
		stat = &FileStat{}

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

		sh.statsCh <- *stat
	}
	done()
}

func (sh *Shrinker) runPrinter() (err error) {
	var wg sync.WaitGroup
	wg.Add(sh.concurentLimit)
	for i := 0; i < sh.concurentLimit; i++ {
		go sh.printer(wg.Done)
	}
	wg.Wait()
	close(sh.statsCh)
	return nil
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
	if sh.isExcludeName(de.Name()) {
		return ExcludeError
	}

	ff := &removeObjInfo{
		isDir:    de.IsDir(),
		filename: de.Name(),
		fullpath: osPathname,
	}

	if sh.isIncludeName(de.Name()) {
		sh.removeCh <- ff
		return nil
	} else if de.IsDir() && sh.isDirToRemove(de.Name()) {
		sh.removeCh <- ff
		return SkipDirError
	} else if de.IsRegular() && sh.isFileToRemove(de.Name()) {
		sh.removeCh <- ff
		return nil
	}
	return NotProcessError
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

type tmpStruct struct {
	tab      string
	tabToAdd string
	filename string
	isDir    bool
	space    int
}

func (sh *Shrinker) layoutPrinter(done func()) {
	var workspacePath []string = make([]string, 0)
	var isLast bool = false
	var tabToAdd string = ""
	var lastPrintLine *tmpStruct

	for removeObjInfo := range sh.removeCh {
		fullPath := strings.ReplaceAll(removeObjInfo.fullpath, sh.checkPath, "")
		isLast = false

		currentPath := strings.Split(fullPath, "/")[1:]
		if len(currentPath) >= len(workspacePath) {
			workspacePath = currentPath[0:]
		} else {
			for i := 0; i < len(workspacePath); i++ {
				if workspacePath[i] != currentPath[i] {
					workspacePath = currentPath[0 : i+1]
					isLast = true
					break
				}
			}
		}

		if isLast {
			tabToAdd = lastChar
		} else {
			tabToAdd = progressChar
		}

		tab := strings.Repeat("│"+tabChar, len(workspacePath))
		if removeObjInfo.isDir {
			if lastPrintLine != nil {
				logLine := fmt.Sprintf("%v%v%v\n", lastPrintLine.tab, tabToAdd, lastPrintLine.filename)
				fmt.Println(logLine)
			}
			lastPrintLine = &tmpStruct{
				filename: removeObjInfo.filename,
				isDir:    true,
				space:    0,
				tab:      tab,
				tabToAdd: tabToAdd,
			}
		} else {
			var fileSize string
			stat, err := fsManager.Stat(removeObjInfo.fullpath, false)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if lastPrintLine.space != 0 {
				fileSize = fmt.Sprintf("%vb", lastPrintLine.space)
			} else {
				fileSize = "empty"
			}
			if lastPrintLine != nil {
				logLine := fmt.Sprintf("%v%v%v (%v)\n", lastPrintLine.tab, tabToAdd, lastPrintLine.filename, fileSize)
				fmt.Println(logLine)
			}
			lastPrintLine = &tmpStruct{
				filename: removeObjInfo.filename,
				isDir:    false,
				space:    int(stat.Size()),
				tab:      tab,
				tabToAdd: tabToAdd,
			}
		}
	}
	if lastPrintLine != nil {
		var fileSize string
		if lastPrintLine.isDir {
			logLine := fmt.Sprintf("%v%v%v\n", lastPrintLine.tab, tabToAdd, lastPrintLine.filename)
			fmt.Println(logLine)
		} else {
			if lastPrintLine.space != 0 {
				fileSize = fmt.Sprintf("%vb", lastPrintLine.space)
			} else {
				fileSize = "empty"
			}
			logLine := fmt.Sprintf("%v%v%v (%v)\n", lastPrintLine.tab, tabToAdd, lastPrintLine.filename, fileSize)
			fmt.Println(logLine)
		}
	}
	done()
}

func (sh *Shrinker) startPrinter() error {
	var wg sync.WaitGroup
	wg.Add(1)
	// TODO: remove hardcode func
	go sh.layoutPrinter(wg.Done)
	_ = walker.Walk(sh.checkPath, func(osPathname string, de FileInfoI) error {
		ff := &removeObjInfo{
			isDir:    de.IsDir(),
			filename: de.Name(),
			fullpath: osPathname,
		}

		sh.removeCh <- ff
		return nil
	}, sh.fileFilterErrCallback)
	close(sh.removeCh)
	wg.Wait()
	return nil
}

func (sh *Shrinker) startCleaner() error {
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
