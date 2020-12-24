package shrink

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"sync"

	. "github.com/icecream78/node_shrinker/fs"
	. "github.com/icecream78/node_shrinker/walker"

	humanize "github.com/dustin/go-humanize"
	color "github.com/logrusorgru/aurora"
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
	concurentLimit int
	checkPath      string
	filter         *Filter
	logger         Logger
}

func NewShrinker(cfg *Config, logger Logger) (*Shrinker, error) {
	if !pathExists(cfg.CheckPath) {
		return nil, NotExistError
	}

	concurentLimit := cfg.ConcurentLimit
	if concurentLimit == 0 {
		concurentLimit = 1
	}
	var checkPath string

	if cfg.CheckPath == "" {
		checkPath, _ = fsManager.Getwd()
	}

	walker = NewDirWalker(cfg.DryRun)

	return &Shrinker{
		verboseOutput:  cfg.VerboseOutput,
		checkPath:      checkPath,
		filter:         NewFilter(cfg.IncludeNames, cfg.ExcludeNames, cfg.RemoveFileExt),
		concurentLimit: concurentLimit,
		logger:         logger,
	}, nil
}

func (sh *Shrinker) DryRun(ctx context.Context) (stats *FileStat) {
	filesCh := sh.layoutPrinterWrapper(sh.checkPath)
	statsCh := sh.runStatGrabber(ctx, filesCh)
	stats = <-statsCh

	return stats
}

func (sh *Shrinker) Clean(ctx context.Context) (stats *FileStat) {
	filesCh := sh.inspectPath(sh.checkPath)
	removeCh := sh.runCleaners(ctx, filesCh)
	statsCh := sh.runStatGrabber(ctx, removeCh)

	stats = <-statsCh
	return stats
}

func (sh *Shrinker) inspectPath(path string) chan *removeObjInfo {
	inspectCh := make(chan *removeObjInfo)
	go func(ch chan *removeObjInfo) {
		defer close(ch)

		_ = walker.Walk(sh.checkPath, sh.fileFilterCallback(inspectCh), sh.fileFilterErrCallback)
	}(inspectCh)

	return inspectCh
}

func (sh *Shrinker) cleaner(ctx context.Context, done func(), removeCh chan *removeObjInfo, statsCh chan *FileStat) {
	var err error
	var stat *FileStat

	for {
		select {
		case obj := <-removeCh:
			if sh.verboseOutput {
				sh.logger.Infof("removing: %s\n", obj.fullpath)
			}

			if obj.isDir {
				stat, err = fsManager.Stat(obj.fullpath, true)
			} else {
				stat, err = fsManager.Stat(obj.fullpath, false)
			}

			if err != nil {
				if sh.verboseOutput {
					sh.logger.Infof("ERROR: %s\n", err)
				}
				continue
			}

			if err = fsManager.RemoveAll(obj.fullpath); err != nil {
				if sh.verboseOutput {
					sh.logger.Infof("ERROR: %s\n", err)
				}
				continue
			}
			statsCh <- stat
		case <-ctx.Done():
			done()
			return
		}
	}
}

func (sh *Shrinker) runCleaners(ctx context.Context, input chan *removeObjInfo) (output chan *FileStat) {
	statsCh := make(chan *FileStat)
	go func(out chan *FileStat) {
		var wg sync.WaitGroup
		wg.Add(sh.concurentLimit)
		for i := 0; i < sh.concurentLimit; i++ {
			go sh.cleaner(ctx, wg.Done, input, out)
		}
		wg.Wait()
		close(out)
	}(statsCh)

	return statsCh
}

func (sh *Shrinker) runStatGrabber(ctx context.Context, statsCh chan *FileStat) chan *FileStat {
	resCh := make(chan *FileStat)

	go func(resCh chan *FileStat) {
		var removedCount int64
		var removedSize int64

		defer close(resCh)
		defer func() {
			resCh <- NewFileStat("result", "result", removedSize, removedCount)
		}()

		for {
			select {
			case stat, isOpen := <-statsCh:
				if stat != nil {
					removedCount += stat.FilesCount()
					removedSize += stat.Size()
				}
				if !isOpen {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}(resCh)

	return resCh
}

func (sh *Shrinker) fileFilterCallback(passCh chan *removeObjInfo) func(string, FileInfoI) error {
	return func(osPathname string, de FileInfoI) error {
		isProcessable, err := sh.filter.Check(de)
		if isProcessable {
			ff := removeObjInfo{
				isDir:    de.IsDir(),
				filename: de.Name(),
				fullpath: osPathname,
			}
			passCh <- &ff
		}

		if err != nil {
			return err
		}
		return nil
	}
}

func (sh *Shrinker) fileFilterErrCallback(osPathname string, err error) ErrorAction {
	// TODO: more informative logging about errors
	if err == SkipDirError {
		return SkipNode
	}

	if sh.verboseOutput {
		sh.logger.Infof("ERROR: %s\n", err)
	}
	return SkipNode
}

func (sh *Shrinker) layoutPrinterWrapper(checkPath string) chan *FileStat {
	ch := make(chan *FileStat)

	go func(ch chan *FileStat) {
		_ = sh.layoutPrinter(checkPath, "", ch)

		close(ch)
	}(ch)

	return ch
}

func (sh *Shrinker) layoutPrinter(checkPath string, tabPassed string, statsCh chan *FileStat) error {
	files, err := ioutil.ReadDir(checkPath)
	if err != nil {
		return err
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
	var fileSize int64
	var fileStat *FileStat

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
				fileStat = stat
			} else {
				fileSize = 0
				fileStat = NewFileStat(file.Name(), path.Join(checkPath, file.Name()), 0, 1)
			}
		} else {
			if isFileInProcess {
				printName = color.Green(printName)
			} else {
				printName = color.Red(printName)
			}
			fileSize = file.Size()
			fileStat = NewFileStat(file.Name(), path.Join(checkPath, file.Name()), fileSize, 1)
		}

		if fileSize != 0 {
			printFileSize = color.Cyan(fmt.Sprintf("%v", humanize.Bytes(uint64(fileSize))))
		} else {
			printFileSize = color.Yellow("empty")
		}

		logLine = fmt.Sprintf("%v%v%v (%v)\n", tabPassed, tabToAdd, printName, printFileSize)
		sh.logger.Infoln(logLine)

		if isFileInProcess {
			statsCh <- fileStat
		}

		// skip directories that matched by name
		if file.IsDir() && !isFileInProcess {
			nextDirPath := fmt.Sprintf("%v/%v", checkPath, file.Name())
			_ = sh.layoutPrinter(nextDirPath, tabToPass, statsCh)
		}
	}

	return nil
}
