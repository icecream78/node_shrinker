package cmd

import (
	"context"
	"errors"
	"os"
	"path"

	"github.com/dustin/go-humanize"
	color "github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"

	"github.com/icecream78/node_shrinker/fs"
	"github.com/icecream78/node_shrinker/shrink"
	"github.com/spf13/cobra"
)

var dryRun, verboseOutput, isNodeDir bool
var checkPath string
var excludeNames, includeNames, includeExtensions []string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "node_shrinker",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New()

		if checkPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal("Fail get current directory")
			}
			checkPath = cwd
		}

		if isNodeDir {
			checkPath = path.Join(checkPath, "node_modules")
		}

		if exists, err := isDirectoryExists(checkPath); err != nil {
			if errors.Is(err, ProvidedFileError) {
				logger.Info("Provided specific file, not a path to directory for clean up. Shut down...")
				return
			}

			logger.Infof("Fail to check path existence with error: %s", err.Error())
			return
		} else if !exists {
			logger.Info("Provided non exist path. Shut down...")
			return
		}

		// TODO: move Shrinker configuring with builder
		shrinker, err := shrink.NewShrinker(&shrink.Config{
			CheckPath:     checkPath,
			VerboseOutput: verboseOutput,
			ExcludeNames:  excludeNames,
			IncludeNames:  includeNames,
			RemoveFileExt: includeExtensions,
		}, logger)

		if err != nil {
			if errors.Is(err, shrink.NotExistError) {
				log.Infof("Path %s doesn`t exist\n", checkPath)
				os.Exit(1)
			}

			log.Infof("Something has broken=) %v\n", err)
			os.Exit(1)
		}

		logger.Infof("Start process directory %s\n", checkPath)

		ctx := context.TODO()

		var stats *fs.FileStat
		if dryRun {
			stats = shrinker.DryRun(ctx)
		} else {
			stats = shrinker.Clean(ctx)
		}

		if err != nil {
			log.Infof("Something has broken2=) %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			logger.Infoln("Dry-run stats:")
			logger.Infof("space to release: %v\n", color.Cyan(humanize.Bytes(uint64(stats.Size()))))
			logger.Infof("files count to remove: %d\n", color.Cyan(stats.FilesCount()))
		} else {
			logger.Infoln("Remove stats:")
			logger.Infof("released space: %v\n", color.Cyan(humanize.Bytes(uint64(stats.Size()))))
			logger.Infof("files count: %d\n", color.Cyan(stats.FilesCount()))
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Infoln(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
	log.SetOutput(os.Stdout)

	rootCmd.PersistentFlags().StringVarP(&checkPath, "dir", "d", "", "path to directory where need cleanup")
	rootCmd.PersistentFlags().StringSliceVarP(&excludeNames, "exclude", "e", []string{}, "list of files/directories that should not be removed. Flag can be specified multiple times. Support regular expression syntax")
	rootCmd.PersistentFlags().StringSliceVarP(&includeNames, "include", "i", []string{}, "list of files/directories that should be included in remove list. Flag can be specified multiple times. Support regular expression syntax")
	rootCmd.PersistentFlags().StringSliceVarP(&includeExtensions, "ext", "x", []string{}, "list of file extensions that should be removed. Flag can be specified multiple times")

	rootCmd.PersistentFlags().BoolVarP(&verboseOutput, "verbose", "v", false, "more detailed output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "display what files will be removed")
	rootCmd.PersistentFlags().BoolVar(&isNodeDir, "node", true, "need detect node_modules dir")
}
