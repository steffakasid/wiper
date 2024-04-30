package wiper

import (
	"os"
	"path"
	"regexp"
	"slices"

	"github.com/steffakasid/eslog"
)

type Wiper struct {
	WipeOut        []string `json:"wipe_out,omitempty" mapstructure:"wipe_out" yaml:"wipe_out"`
	WipeOutPattern []string `json:"wipe_out_pattern,omitempty"  mapstructure:"wipe_out_pattern"  yaml:"wipe_out_pattern"`
	ExcludeDir     []string `json:"exclude_dir,omitempty" mapstructure:"exclude_dir" yaml:"exclude_dir"`
	BaseDir        string   `json:"base_dir,omitempty" mapstructure:"base_dir" yaml:"base_dir"`
	UseTrash       bool     `json:"use_trash,omitempty" mapstructure:"use_trash" yaml:"use_trash"`
}

func GetInstance() *Wiper {
	if wiper == nil {
		panic("Wiper object not initialized!")
	}
	return wiper
}

func (w Wiper) WipeFiles(dir string) error {

	if dir == "" {
		dir = w.BaseDir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	trash := path.Join(home, ".Trash")
	if w.UseTrash {
		if _, err := os.Stat(trash); err != nil {
			eslog.Debugf("%s not existing creating it.")
			err := os.Mkdir(trash, 0700)
			if err != nil {
				return err
			}
		}
	}

	if entries, err := os.ReadDir(dir); err != nil {
		return err
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				if !slices.Contains(w.ExcludeDir, entry.Name()) {
					return w.WipeFiles(path.Join(dir, entry.Name()))
				}
				eslog.Debug("Exclude dir", entry.Name())
			} else {
				if slices.Contains(w.WipeOut, entry.Name()) || slices.ContainsFunc(w.WipeOutPattern, func(pattern string) bool {
					matcher, err := regexp.Compile(pattern)
					eslog.LogIfError(err, eslog.Fatal)

					return matcher.Match([]byte(entry.Name()))
				}) {
					if w.UseTrash {
						err := os.Rename(path.Join(dir, entry.Name()), path.Join(trash, entry.Name()))
						if err != nil {
							return err
						}
					} else {
						err := os.Remove(path.Join(dir, entry.Name()))
						if err != nil {
							return err
						}
					}
				} else {
					eslog.Debug("Skipping", entry.Name())
				}
			}
		}
	}
	return nil
}
