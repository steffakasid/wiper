/*
Copyright © 2024 steffakasid
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/steffakasid/eslog"
	wiper "github.com/steffakasid/wiper/internal"
)

// Constants used in command flags
const (
	wipeOutFlag        = "wipe_out"
	wipeOutPatternFlag = "wipe_out_pattern"
	excludeDirFlag     = "exclude_dir"
	excludeFileFlag    = "exclude_file"
	baseDirFlag        = "base_dir"
	useTrashFlag       = "use_trash"
	configFlag         = "config"
	debugFlag          = "debug"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wiper",
	Short: "Wiper is a tool to wipe out files.",
	Long:  `Wiper is a tool to wipe out files. Like e.g. *.orig files created by editors.`,
	RunE:  RunWiperE,
}

func RunWiperE(cmd *cobra.Command, args []string) error {

	if viper.GetBool(debugFlag) {
		err := eslog.Logger.SetLogLevel("debug")
		eslog.LogIfError(err, eslog.Error)
		eslog.Info("Debugging enabled.")
	} else {
		err := eslog.Logger.SetLogLevel("info")
		eslog.LogIfError(err, eslog.Error)
		eslog.Info("Debugging disabled.")
	}

	wiper := wiper.GetInstance()
	errChan := make(chan error)
	wiper.WipeFiles(nil, "", errChan)
	if len(errChan) > 0 {
		for err := range errChan {
			eslog.Error(err)
		}
		return errors.New("errors occurred during wiping files")
	}
	fmt.Printf("Inspected %d files and wiped %d files.\n", wiper.InspectedFiles, wiper.WipedFiles)
	fmt.Printf("Inspected %d directories and wiped %d directories.\n", wiper.InspectedDirs, wiper.WipedDirs)
	return nil
}

func Execute(version string) {
	rootCmd.Version = version
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	err := eslog.Logger.SetLogLevel("debug")
	eslog.LogIfError(err, eslog.Error)

	cobra.OnInitialize(wiper.InitConfig)

	peristentFlags := rootCmd.PersistentFlags()
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	peristentFlags.StringP(baseDirFlag, "b", home, "Base dir to scan files to be wiped out.")
	peristentFlags.StringArrayP(excludeDirFlag, "e", []string{}, "String array of excluded directories.")
	peristentFlags.StringArrayP(excludeFileFlag, "f", []string{}, "String array of excluded files.")
	peristentFlags.StringArrayP(wipeOutFlag, "w", []string{}, "String array of files to be wiped.")
	peristentFlags.StringArrayP(wipeOutPatternFlag, "p", []string{}, "String array of patterns for files to be wiped.")
	peristentFlags.BoolP(useTrashFlag, "t", false, "Enable using trash folder ($HOME/.Trash). If folder does not exist already, it will be created. [default: false]")
	peristentFlags.BoolP(debugFlag, "d", false, "Enable debugging.")
	peristentFlags.StringVar(&wiper.CfgFile, configFlag, "", "Config file to use insted default: $HOME/.config/wiper/config")

	cobra.CheckErr(viper.BindPFlags(peristentFlags))
}
