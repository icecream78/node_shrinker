package shrunk

import (
	"fmt"
	"os"

	"github.com/karrick/godirwalk"
)

type Shrunker struct {
	cfg            *Config
	shrunkDirNames map[string]struct{}
}

func NewShrunker(cfg *Config) *Shrunker {
	return &Shrunker{
		cfg:            cfg,
		shrunkDirNames: sliceToMap(cfg.RemoveDirNames),
	}
}

func (sh *Shrunker) Start() error {
	return sh.start()
}

func (sh *Shrunker) start() error {
	err := godirwalk.Walk(sh.cfg.CheckPath, &godirwalk.Options{
		Unsorted: true, // for higher speed walking dir tree
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if _, exists := sh.shrunkDirNames[de.Name()]; exists {
				fmt.Printf("%s %s\n", de.ModeType(), osPathname)
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return err
}
