package wiper

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

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
	trashMu            sync.Mutex
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
	eslog.Debugf("CurrentDir %s", dir)
	w.mu.Lock()
	w.InspectedDirs++
	w.mu.Unlock()

	if wg == nil {
		wg = &sync.WaitGroup{}
		defer func() {
			wg.Wait()
			close(errChan)
		}()
	}

	trash := initTrash(w)

	entries, err := os.ReadDir(dir)
	if err != nil {
		errChan <- err
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			w.handleDir(wg, dir, trash, name, errChan)
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

func (w *Wiper) handleDir(wg *sync.WaitGroup, dir, trash, name string, errChan chan error) {
	if slices.Contains(w.ExcludeDir, name) {
		return
	}
	if w.shouldWipe(name, true) {
		w.mu.Lock()
		w.WipedDirs++
		w.mu.Unlock()
		target := path.Join(dir, name)
		var err error
		if w.UseTrash {
			err = w.moveToTrash(target, trash, true)
		} else {
			err = os.RemoveAll(target)
		}
		if err != nil {
			errChan <- err
		}
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
		err = w.moveToTrash(path.Join(dir, name), trash, false)
	} else {
		err = os.Remove(path.Join(dir, name))
	}
	if err != nil {
		errChan <- err
	}
}

func (w *Wiper) moveToTrash(sourcePath, trash string, isDir bool) error {
	w.trashMu.Lock()
	defer w.trashMu.Unlock()

	return os.Rename(sourcePath, uniqueTrashDestination(trash, filepath.Base(sourcePath), isDir))
}

func uniqueTrashDestination(trash, name string, isDir bool) string {
	destination := filepath.Join(trash, name)
	if !pathExists(destination) {
		return destination
	}

	timestamp := time.Now().Format("20060102-150405.000000000")
	for attempt := 0; ; attempt++ {
		suffix := timestamp
		if attempt > 0 {
			suffix = fmt.Sprintf("%s-%d", timestamp, attempt)
		}

		candidate := filepath.Join(trash, trashNameWithPostfix(name, suffix, isDir))
		if !pathExists(candidate) {
			return candidate
		}
	}
}

func trashNameWithPostfix(name, postfix string, isDir bool) string {
	if isDir {
		return fmt.Sprintf("%s-%s", name, postfix)
	}

	ext := filepath.Ext(name)
	if ext == "" || len(ext) == len(name) {
		return fmt.Sprintf("%s-%s", name, postfix)
	}

	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s-%s%s", base, postfix, ext)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func dirExists(path string) bool {
	return pathExists(path)
}

func (w *Wiper) matchWipe(name string, items, patterns, exclude []string) bool {
	if !slices.Contains(exclude, name) {
		if slices.Contains(items, name) {
			return true
		}
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
