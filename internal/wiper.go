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
	WipeOut            []string `json:"wipe_out,omitempty" mapstructure:"wipe_out" yaml:"wipe_out"`
	WipeOutPattern     []string `json:"wipe_out_pattern,omitempty"  mapstructure:"wipe_out_pattern"  yaml:"wipe_out_pattern"`
	WipeOutDirs        []string `json:"wipe_out_dirs,omitempty" mapstructure:"wipe_out_dirs" yaml:"wipe_out_dirs"`
	WipeOutPatternDirs []string `json:"wipe_out_pattern_dirs,omitempty" mapstructure:"wipe_out_pattern_dirs" yaml:"wipe_out_pattern_dirs"`
	ExcludeFile        []string `json:"exclude_file,omitempty" mapstructure:"exclude_file" yaml:"exclude_file"`
	ExcludeDir         []string `json:"exclude_dir,omitempty" mapstructure:"exclude_dir" yaml:"exclude_dir"`
	BaseDir            string   `json:"base_dir,omitempty" mapstructure:"base_dir" yaml:"base_dir"`
	UseTrash           bool     `json:"use_trash,omitempty" mapstructure:"use_trash" yaml:"use_trash"`
	InspectedFiles     int      `json:"-"`
	WipedFiles         int      `json:"-"`
	InspectedDirs      int      `json:"-"`
	WipedDirs          int      `json:"-"`
	mu                 sync.Mutex
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
	w.mu.Lock()
	w.InspectedDirs++
	w.mu.Unlock()

	trash := initTrash(w)

	entries, err := os.ReadDir(dir)
	if err != nil {
		errChan <- err
		return
	}

	if wg == nil {
		wg = &sync.WaitGroup{}
		defer func() {
			wg.Wait()
			close(errChan)
		}()
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

func initTrash(w *Wiper) string {
	home, _ := os.UserHomeDir()
	trash := path.Join(home, ".Trash")
	if w.UseTrash && !dirExists(trash) {
		_ = os.Mkdir(trash, 0700)
	}
	return trash
}

func (w *Wiper) handleDir(wg *sync.WaitGroup, dir, name string, errChan chan error) {
	if slices.Contains(w.ExcludeDir, name) {
		return
	}
	if w.shouldWipe(name, true) {
		w.mu.Lock()
		w.WipedDirs++
		w.mu.Unlock()
		err := os.RemoveAll(path.Join(dir, name))
		eslog.LogIfError(err, eslog.Error)
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

	if !w.shouldWipe(name, false) {
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

func (w *Wiper) matchWipe(name string, items, patterns, exclude []string) bool {
	if slices.Contains(items, name) {
		return true
	}
	if !slices.Contains(exclude, name) {
		for _, pattern := range patterns {
			matcher, err := regexp.Compile(pattern)
			eslog.LogIfError(err, eslog.Fatal)
			if matcher.MatchString(name) {
				return true
			}
		}
	}
	return false
}

func (w *Wiper) shouldWipe(name string, isDir bool) bool {
	if isDir {
		return w.matchWipe(name, w.WipeOutDirs, w.WipeOutPatternDirs, w.ExcludeDir)
	}

	return w.matchWipe(name, w.WipeOut, w.WipeOutPattern, w.ExcludeFile)
}
