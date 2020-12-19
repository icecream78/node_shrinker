package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"

	shrunk "github.com/icecream78/node_shrinker/shrink"
	"github.com/spf13/cobra"
)

var dryRun bool
var verboseOutput bool
var checkPath string
var excludeNames []string
var includeNames []string
var includeExtensions []string

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
		// TODO: move Shrinker configuring with builder
		shrinker, err := shrunk.NewShrinker(&shrunk.Config{
			CheckPath:     checkPath,
			VerboseOutput: verboseOutput,
			ExcludeNames:  excludeNames,
			IncludeNames:  includeNames,
			RemoveFileExt: includeExtensions,
			DryRun:        dryRun,
		}, log.New())
		if err != nil {
			log.Infof("Something has broken=) %v\n", err)
			os.Exit(1)
		}
		err = shrinker.Start()
		if err != nil {
			log.Infof("Something has broken2=) %v\n", err)
			os.Exit(1)
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
}
