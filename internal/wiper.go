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
					err := os.Remove(path.Join(dir, entry.Name()))
					return err
				} else {
					eslog.Debug("Skipping", entry.Name())
				}
			}
		}
	}
	return nil
}
