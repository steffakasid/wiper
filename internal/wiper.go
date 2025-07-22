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
	ExcludeFile    []string `json:"exclude_file,omitempty" mapstructure:"exclude_file" yaml:"exclude_file"`
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

	home, _ := os.UserHomeDir()
	trash := path.Join(home, ".Trash")
	if w.UseTrash && !dirExists(trash) {
		_ = os.Mkdir(trash, 0700)
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
		name := entry.Name()
		if entry.IsDir() {
			w.handleDir(wg, dir, name, errChan)
		} else {
			w.handleFile(dir, trash, name, errChan)
		}
	}
}

func (w *Wiper) handleDir(wg *sync.WaitGroup, dir, name string, errChan chan error) {
	if slices.Contains(w.ExcludeDir, name) {
		return
	}
	wg.Add(1)
	go func(subDir string) {
		defer wg.Done()
		w.WipeFiles(wg, subDir, errChan)
	}(path.Join(dir, name))
}

func (w *Wiper) handleFile(dir, trash, name string, errChan chan error) {
	w.mu.Lock()
	w.InspectedFiles++
	w.mu.Unlock()

	if !w.shouldWipe(name) {
		return
	}

	w.mu.Lock()
	w.WipedFiles++
	w.mu.Unlock()

	var err error
	if w.UseTrash {
		err = os.Rename(path.Join(dir, name), path.Join(trash, name))
	} else {
		err = os.Remove(path.Join(dir, name))
	}
	if err != nil {
		errChan <- err
	}
}

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (w *Wiper) shouldWipe(name string) bool {
	if slices.Contains(w.WipeOut, name) {
		return true
	} else if !slices.Contains(w.ExcludeFile, name) {
		for _, pattern := range w.WipeOutPattern {
			matcher, err := regexp.Compile(pattern)
			eslog.LogIfError(err, eslog.Fatal)
			if matcher.MatchString(name) {
				return true
			}
		}
	}
	return false
}
