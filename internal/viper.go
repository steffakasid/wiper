package wiper

import (
	"bytes"
	"fmt"
	"os"
	"path"

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

	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath(path.Join(home, ".config", "wiper"))
		viper.SetConfigType("yaml")
		viper.SetConfigName(".wiper")
	}

	viper.AutomaticEnv()

	usedConfigFile := getConfigFilename(home)
	if usedConfigFile != "" {
		cleartext, err := decrypt.File(usedConfigFile, configFileType)

		if err != nil {
			eslog.Warnf("Error decrypting. %s. Maybe you're not using an encrypted config?", err)
			if err := viper.ReadInConfig(); err != nil {
				eslog.Warnf("Error reading config. %s. Are you using a config?", err)
			} else {
				eslog.Debug("Using config file:", viper.ConfigFileUsed())
			}
		} else {
			if err := viper.ReadConfig(bytes.NewBuffer(cleartext)); err != nil {
				eslog.Fatal(err)
			} else {
				eslog.Debug("Using sops encrypted config file:", viper.ConfigFileUsed())
			}
		}
	} else {
		eslog.Debug("No config file used!")
	}
	fileWiper := &Wiper{}
	err = viper.UnmarshalExact(fileWiper)
	eslog.LogIfError(err, eslog.Fatal)
}

func getConfigFilename(homedir string) string {
	pathWithoutExt := path.Join(homedir, configFileName)
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
