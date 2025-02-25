package wiper

import (
	"os"
	"path"
	"regexp"
	"slices"
	"sync"

	"github.com/steffakasid/eslog"
)

type Wiper struct {
	WipeOut        []string `json:"wipe_out,omitempty" mapstructure:"wipe_out" yaml:"wipe_out"`
	WipeOutPattern []string `json:"wipe_out_pattern,omitempty"  mapstructure:"wipe_out_pattern"  yaml:"wipe_out_pattern"`
	ExcludeDir     []string `json:"exclude_dir,omitempty" mapstructure:"exclude_dir" yaml:"exclude_dir"`
	BaseDir        string   `json:"base_dir,omitempty" mapstructure:"base_dir" yaml:"base_dir"`
	UseTrash       bool     `json:"use_trash,omitempty" mapstructure:"use_trash" yaml:"use_trash"`
	InspectedFiles int      `json:"-"`
	WipedFiles     int      `json:"-"`
	mu             sync.Mutex
}

func GetInstance() *Wiper {
	if wiper == nil {
		panic("Wiper object not initialized!")
	}
	return wiper
}

func (w *Wiper) WipeFiles(wg *sync.WaitGroup, dir string, errChan chan error) {
	if dir == "" {
		dir = w.BaseDir
	}
	eslog.Debug("CurrentDir", dir)

	home, err := os.UserHomeDir()
	if err != nil {
		errChan <- err
	}
	trash := path.Join(home, ".Trash")
	if w.UseTrash && !dirExists(trash) {
		eslog.Debugf("%s not existing creating it.", trash)
		if err := os.Mkdir(trash, 0700); err != nil {
			errChan <- err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		errChan <- err
		return
	}

	if wg == nil {
		wg = &sync.WaitGroup{}
		defer wg.Wait()
		close(errChan)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if slices.Contains(w.ExcludeDir, entry.Name()) {
				eslog.Debug("Exclude dir", entry.Name())
				continue
			}
			wg.Add(1)
			go func(subDir string) {
				defer wg.Done()
				w.WipeFiles(wg, subDir, errChan)
			}(path.Join(dir, entry.Name()))
		} else {
			w.mu.Lock()
			w.InspectedFiles++
			w.mu.Unlock()

			if w.shouldWipe(entry.Name()) {
				w.mu.Lock()
				w.WipedFiles++
				w.mu.Unlock()
				if w.UseTrash {
					err := os.Rename(path.Join(dir, entry.Name()), path.Join(trash, entry.Name()))
					if err != nil {
						errChan <- err
					}
				} else {
					err := os.Remove(path.Join(dir, entry.Name()))
					if err != nil {
						errChan <- err
					}
				}
			} else {
				eslog.Debug("Skipping", entry.Name())
			}
		}
	}
}

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (w *Wiper) shouldWipe(name string) bool {
	if slices.Contains(w.WipeOut, name) {
		return true
	}
	for _, pattern := range w.WipeOutPattern {
		matcher, err := regexp.Compile(pattern)
		eslog.LogIfError(err, eslog.Fatal)
		if matcher.MatchString(name) {
			return true
		}
	}
	return false
}
