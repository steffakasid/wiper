package wiper

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/getsops/sops/v3/decrypt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/steffakasid/eslog"
)

const (
	configFileType = "yaml"
	configFileName = "config"
)

var wiper *Wiper
var CfgFile string

func InitConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	configPath := path.Join(home, ".config", "wiper", configFileName)
	if CfgFile != "" {
		configPath = CfgFile
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath(filepath.Dir(configPath))
		viper.SetConfigType("yaml")
		viper.SetConfigName(configFileName)
	}

	viper.AutomaticEnv()

	usedConfigFile := getConfigFilename(configPath)
	if usedConfigFile != "" {
		cleartext, err := decrypt.File(usedConfigFile, configFileType)

		if err != nil {
			eslog.Debugf("Error decrypting. %s", err)
			if err := viper.ReadInConfig(); err != nil {
				eslog.Warnf("Error reading config. %s.", err)
			} else {
				eslog.Debug("Using config file:", usedConfigFile)
			}
		} else {
			if err := viper.ReadConfig(bytes.NewBuffer(cleartext)); err != nil {
				eslog.Fatal(err)
			} else {
				eslog.Debug("Using sops encrypted config file:", usedConfigFile)
			}
		}
	} else {
		eslog.Debug("No config file used!")
	}
	wiper = &Wiper{}
	err = viper.Unmarshal(wiper)
	eslog.LogIfError(err, eslog.Fatal)
}

func getConfigFilename(pathWithoutExt string) string {

	eslog.Debugf("Check if %s exists", pathWithoutExt)
	if _, err := os.Stat(pathWithoutExt); err == nil {
		return pathWithoutExt
	}

	pathWithExt := fmt.Sprintf("%s.%s", pathWithoutExt, configFileType)
	eslog.Debugf("Check if %s exists", pathWithExt)
	if _, err := os.Stat(pathWithExt); err == nil {
		return pathWithExt
	}
	pathWithExt = fmt.Sprintf("%s.%s", pathWithoutExt, "yml")
	eslog.Debugf("Check if %s exists", pathWithExt)
	if _, err := os.Stat(pathWithExt); err == nil {
		return pathWithExt
	}
	return ""
}
