package wiper

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findTrashEntry(t *testing.T, trashDir, prefix, suffix string) string {
	t.Helper()

	entries, err := os.ReadDir(trashDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			return filepath.Join(trashDir, name)
		}
	}

	t.Fatalf("no trash entry found with prefix %q and suffix %q", prefix, suffix)
	return ""
}

func TestWipeFiles(t *testing.T) {

	receiveAllErrors := func(errChan chan error) []error {
		errs := []error{}
		for err := range errChan {
			if err != nil {
				errs = append(errs, err)
			}
		}
		return errs
	}

	t.Run("Success", func(t *testing.T) {
		testDir := t.TempDir()
		fileToDelete, err := os.CreateTemp(testDir, "")
		require.NoError(t, err)
		err = fileToDelete.Close()
		require.NoError(t, err)
		sut := Wiper{
			WipeOut: []string{filepath.Base(fileToDelete.Name())},
			BaseDir: testDir,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, fileToDelete.Name())
	})

	t.Run("Success with pattern", func(t *testing.T) {
		testdir := t.TempDir()
		pattern := `.*\\.orig`
		fileToDelete, err := os.CreateTemp(testdir, pattern)
		require.NoError(t, err)
		err = fileToDelete.Close()
		require.NoError(t, err)

		sut := Wiper{
			WipeOutPattern: []string{pattern},
			BaseDir:        testdir,
		}
		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, fileToDelete.Name())
	})

	t.Run("Do not delete all", func(t *testing.T) {
		testDir := t.TempDir()
		fileToDelete, err := os.CreateTemp(testDir, "")
		require.NoError(t, err)
		fileNotToDelete, err := os.CreateTemp(testDir, "do-not-delete")
		require.NoError(t, err)
		err = fileToDelete.Close()
		require.NoError(t, err)
		sut := Wiper{
			WipeOut: []string{filepath.Base(fileToDelete.Name())},
			BaseDir: testDir,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, fileToDelete.Name())
		assert.FileExists(t, fileNotToDelete.Name())
	})

	t.Run("Exclude dir and skip files", func(t *testing.T) {
		testDir := t.TempDir()
		fileToDelete, err := os.CreateTemp(testDir, "")
		require.NoError(t, err)
		skippedDir, err := os.MkdirTemp(testDir, "")
		require.NoError(t, err)
		skippedFile, err := os.Create(filepath.Join(skippedDir, filepath.Base(fileToDelete.Name())))
		require.NoError(t, err)
		err = skippedFile.Close()
		require.NoError(t, err)

		sut := Wiper{
			WipeOut:    []string{filepath.Base(fileToDelete.Name())},
			ExcludeDir: []string{filepath.Base(skippedDir)},
			BaseDir:    testDir,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, fileToDelete.Name())
		assert.FileExists(t, skippedFile.Name())
	})

	t.Run("Exclude file", func(t *testing.T) {
		testDir := t.TempDir()
		fileToExclude, err := os.CreateTemp(testDir, "")
		require.NoError(t, err)

		sut := Wiper{
			ExcludeFile: []string{fileToExclude.Name()},
			BaseDir:     testDir,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.FileExists(t, fileToExclude.Name())
	})

	t.Run("UseTrash moves file to trash", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		testDir := filepath.Join(testHome, "source")
		require.NoError(t, os.Mkdir(testDir, 0o755))
		fileToDelete, err := os.CreateTemp(testDir, "testfile")
		require.NoError(t, err)
		require.NoError(t, fileToDelete.Close())

		sut := Wiper{
			WipeOut:  []string{filepath.Base(fileToDelete.Name())},
			BaseDir:  testDir,
			UseTrash: true,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, fileToDelete.Name())
		trashPath := filepath.Join(testHome, ".Trash", filepath.Base(fileToDelete.Name()))
		assert.FileExists(t, trashPath)
	})

	t.Run("UseTrash suffixes duplicate file names", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		testDir := filepath.Join(testHome, "source")
		require.NoError(t, os.Mkdir(testDir, 0o755))

		trashDir := filepath.Join(testHome, ".Trash")
		require.NoError(t, os.Mkdir(trashDir, 0o755))

		fileName := "duplicate.txt"
		existingTrashFile := filepath.Join(trashDir, fileName)
		require.NoError(t, os.WriteFile(existingTrashFile, []byte("existing"), 0o644))

		sourceFile := filepath.Join(testDir, fileName)
		require.NoError(t, os.WriteFile(sourceFile, []byte("fresh"), 0o644))

		sut := Wiper{
			WipeOut:  []string{fileName},
			BaseDir:  testDir,
			UseTrash: true,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.NoFileExists(t, sourceFile)

		content, err := os.ReadFile(existingTrashFile)
		require.NoError(t, err)
		assert.Equal(t, "existing", string(content))

		suffixedPath := findTrashEntry(t, trashDir, "duplicate-", ".txt")
		content, err = os.ReadFile(suffixedPath)
		require.NoError(t, err)
		assert.Equal(t, "fresh", string(content))
	})

	t.Run("Wipe directory", func(t *testing.T) {
		testDir := t.TempDir()
		subDir := filepath.Join(testDir, "todelete")
		require.NoError(t, os.Mkdir(subDir, 0o755))
		f, err := os.Create(filepath.Join(subDir, "inside.txt"))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		sut := Wiper{
			WipeOutDirs: []string{"todelete"},
			BaseDir:     testDir,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.False(t, dirExists(subDir))
	})

	t.Run("UseTrash suffixes duplicate directory names", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		testDir := filepath.Join(testHome, "source")
		require.NoError(t, os.Mkdir(testDir, 0o755))

		trashDir := filepath.Join(testHome, ".Trash")
		require.NoError(t, os.Mkdir(trashDir, 0o755))

		existingTrashDir := filepath.Join(trashDir, "todelete")
		require.NoError(t, os.Mkdir(existingTrashDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(existingTrashDir, "old.txt"), []byte("existing"), 0o644))

		sourceDir := filepath.Join(testDir, "todelete")
		require.NoError(t, os.Mkdir(sourceDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "new.txt"), []byte("fresh"), 0o644))

		sut := Wiper{
			WipeOutDirs: []string{"todelete"},
			BaseDir:     testDir,
			UseTrash:    true,
		}

		errChan := make(chan error)
		sut.WipeFiles(nil, "", errChan)
		errs := receiveAllErrors(errChan)
		assert.Empty(t, errs)
		assert.False(t, dirExists(sourceDir))
		assert.FileExists(t, filepath.Join(existingTrashDir, "old.txt"))

		suffixedDir := findTrashEntry(t, trashDir, "todelete-", "")
		assert.FileExists(t, filepath.Join(suffixedDir, "new.txt"))
	})
}

func TestDirExists(t *testing.T) {
	t.Run("green case - directory exists", func(t *testing.T) {
		testDir := t.TempDir()
		result := dirExists(testDir)
		assert.True(t, result, "dirExists should return true for existing directory")
	})

	t.Run("red case - directory does not exist", func(t *testing.T) {
		result := dirExists("/nonexistent/path/that/should/not/exist")
		assert.False(t, result, "dirExists should return false for nonexistent path")
	})

	t.Run("file exists returns true", func(t *testing.T) {
		testDir := t.TempDir()
		file, err := os.CreateTemp(testDir, "test")
		require.NoError(t, err)
		require.NoError(t, file.Close())

		result := dirExists(file.Name())
		assert.True(t, result, "dirExists should return true for existing file")
	})
}

func TestMatchWipe(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		items    []string
		patterns []string
		exclude  []string
		expected bool
	}{
		{
			name:     "green case - exact match in items",
			itemName: "test.orig",
			items:    []string{"test.orig"},
			patterns: []string{},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "red case - no match",
			itemName: "keep.txt",
			items:    []string{"delete.txt"},
			patterns: []string{},
			exclude:  []string{},
			expected: false,
		},
		{
			name:     "pattern match",
			itemName: "file.orig",
			items:    []string{},
			patterns: []string{`.*\.orig$`},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "multiple patterns - match second",
			itemName: "backup.bak",
			items:    []string{},
			patterns: []string{`.*\.orig$`, `.*\.bak$`},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "excluded item overrides items match",
			itemName: "test.orig",
			items:    []string{"test.orig"},
			patterns: []string{},
			exclude:  []string{"test.orig"},
			expected: false,
		},
		{
			name:     "excluded item overrides pattern match",
			itemName: "file.orig",
			items:    []string{},
			patterns: []string{`.*\.orig$`},
			exclude:  []string{"file.orig"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := Wiper{}
			result := sut.matchWipe(tt.itemName, tt.items, tt.patterns, tt.exclude)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldWipe(t *testing.T) {
	t.Run("green case - file should be wiped", func(t *testing.T) {
		sut := Wiper{
			WipeOut: []string{"test.txt"},
		}
		result := sut.shouldWipe("test.txt", false)
		assert.True(t, result)
	})

	t.Run("red case - file should not be wiped", func(t *testing.T) {
		sut := Wiper{
			WipeOut: []string{"delete.txt"},
		}
		result := sut.shouldWipe("keep.txt", false)
		assert.False(t, result)
	})

	t.Run("green case - directory should be wiped", func(t *testing.T) {
		sut := Wiper{
			WipeOutDirs: []string{"tempdir"},
		}
		result := sut.shouldWipe("tempdir", true)
		assert.True(t, result)
	})

	t.Run("red case - directory should not be wiped", func(t *testing.T) {
		sut := Wiper{
			WipeOutDirs: []string{"tempdir"},
		}
		result := sut.shouldWipe("importantdir", true)
		assert.False(t, result)
	})

	t.Run("file pattern match", func(t *testing.T) {
		sut := Wiper{
			WipeOutPattern: []string{`.*\.orig$`},
		}
		result := sut.shouldWipe("file.orig", false)
		assert.True(t, result)
	})

	t.Run("directory pattern match", func(t *testing.T) {
		sut := Wiper{
			WipeOutPatternDirs: []string{`^\..*$`},
		}
		result := sut.shouldWipe(".hidden", true)
		assert.True(t, result)
	})

	t.Run("excluded file", func(t *testing.T) {
		sut := Wiper{
			WipeOut:     []string{"file.txt"},
			ExcludeFile: []string{"file.txt"},
		}
		result := sut.shouldWipe("file.txt", false)
		assert.False(t, result)
	})

	t.Run("excluded directory", func(t *testing.T) {
		sut := Wiper{
			WipeOutDirs: []string{"dir"},
			ExcludeDir:  []string{"dir"},
		}
		result := sut.shouldWipe("dir", true)
		assert.False(t, result)
	})
}

func TestInitTrash(t *testing.T) {
	t.Run("green case - UseTrash enabled creates trash folder", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		sut := Wiper{
			UseTrash: true,
		}

		result := initTrash(&sut)
		trashPath := filepath.Join(testHome, ".Trash")
		assert.Equal(t, trashPath, result)
		assert.True(t, dirExists(trashPath))
	})

	t.Run("green case - UseTrash disabled returns path without creating", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		sut := Wiper{
			UseTrash: false,
		}

		result := initTrash(&sut)
		trashPath := filepath.Join(testHome, ".Trash")
		assert.Equal(t, trashPath, result)
		assert.False(t, dirExists(trashPath), "trash folder should not be created when UseTrash is false")
	})

	t.Run("trash folder already exists", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)
		trashPath := filepath.Join(testHome, ".Trash")
		require.NoError(t, os.Mkdir(trashPath, 0o700))

		sut := Wiper{
			UseTrash: true,
		}

		result := initTrash(&sut)
		assert.Equal(t, trashPath, result)
		assert.True(t, dirExists(trashPath))
	})
}

func TestHandleDir(t *testing.T) {
	t.Run("green case - directory wiped when matching", func(t *testing.T) {
		testDir := t.TempDir()
		subDir := filepath.Join(testDir, "todelete")
		require.NoError(t, os.Mkdir(subDir, 0o755))

		sut := Wiper{
			WipeOutDirs: []string{"todelete"},
		}

		var wg sync.WaitGroup
		errChan := make(chan error)
		go func() {
			for range errChan {
			}
		}()

		sut.handleDir(&wg, testDir, filepath.Join(testDir, ".Trash"), "todelete", errChan)
		wg.Wait()
		close(errChan)

		assert.False(t, dirExists(subDir))
		assert.Equal(t, 1, sut.WipedDirs)
	})

	t.Run("green case - directory moved to trash when UseTrash enabled", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		testDir := filepath.Join(testHome, "source")
		require.NoError(t, os.Mkdir(testDir, 0o755))

		subDir := filepath.Join(testDir, "todelete")
		require.NoError(t, os.Mkdir(subDir, 0o755))

		trashDir := filepath.Join(testHome, ".Trash")
		require.NoError(t, os.Mkdir(trashDir, 0o755))

		sut := Wiper{
			WipeOutDirs: []string{"todelete"},
			UseTrash:    true,
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		sut.handleDir(&wg, testDir, trashDir, "todelete", errChan)
		wg.Wait()

		assert.False(t, dirExists(subDir))
		assert.True(t, dirExists(filepath.Join(trashDir, "todelete")))
		assert.Equal(t, 1, sut.WipedDirs)
	})

	t.Run("green case - directory skipped when excluded", func(t *testing.T) {
		testDir := t.TempDir()
		subDir := filepath.Join(testDir, "keepdir")
		require.NoError(t, os.Mkdir(subDir, 0o755))

		sut := Wiper{
			ExcludeDir: []string{"keepdir"},
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		sut.handleDir(&wg, testDir, filepath.Join(testDir, ".Trash"), "keepdir", errChan)

		assert.True(t, dirExists(subDir), "directory should not be deleted when excluded")
		assert.Equal(t, 0, sut.WipedDirs)
	})

	t.Run("red case - directory not wiped when not matching", func(t *testing.T) {
		testDir := t.TempDir()
		subDir := filepath.Join(testDir, "keepdir")
		require.NoError(t, os.Mkdir(subDir, 0o755))

		sut := Wiper{
			WipeOutDirs: []string{"todelete"},
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		sut.handleDir(&wg, testDir, filepath.Join(testDir, ".Trash"), "keepdir", errChan)
		wg.Wait()

		assert.True(t, dirExists(subDir), "directory should not be deleted when not matching")
		assert.Equal(t, 0, sut.WipedDirs)
	})
}

func TestHandleFile(t *testing.T) {
	t.Run("green case - file deleted when matching", func(t *testing.T) {
		testDir := t.TempDir()
		file, err := os.CreateTemp(testDir, "todelete")
		require.NoError(t, err)
		require.NoError(t, file.Close())

		sut := Wiper{
			WipeOut: []string{filepath.Base(file.Name())},
		}

		errChan := make(chan error, 10)
		sut.handleFile(testDir, filepath.Join(testDir, ".Trash"), filepath.Base(file.Name()), errChan)

		assert.NoFileExists(t, file.Name())
		assert.Equal(t, 1, sut.WipedFiles)
	})

	t.Run("green case - file moved to trash when UseTrash enabled", func(t *testing.T) {
		testDir := t.TempDir()
		file, err := os.CreateTemp(testDir, "todelete")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		fileName := filepath.Base(file.Name())

		trash := filepath.Join(testDir, ".Trash")
		require.NoError(t, os.Mkdir(trash, 0o755))

		sut := Wiper{
			WipeOut:  []string{fileName},
			UseTrash: true,
		}

		errChan := make(chan error, 10)
		sut.handleFile(testDir, trash, fileName, errChan)

		assert.NoFileExists(t, file.Name())
		assert.FileExists(t, filepath.Join(trash, fileName))
		assert.Equal(t, 1, sut.WipedFiles)
	})

	t.Run("red case - file not deleted when not matching", func(t *testing.T) {
		testDir := t.TempDir()
		file, err := os.CreateTemp(testDir, "keep")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		fileName := filepath.Base(file.Name())

		sut := Wiper{
			WipeOut: []string{"todelete.txt"},
		}

		errChan := make(chan error, 10)
		sut.handleFile(testDir, filepath.Join(testDir, ".Trash"), fileName, errChan)

		assert.FileExists(t, file.Name())
		assert.Equal(t, 0, sut.WipedFiles)
	})

	t.Run("error handling - delete error reported", func(t *testing.T) {
		testDir := t.TempDir()
		readOnlyDir := filepath.Join(testDir, "readonly")
		require.NoError(t, os.Mkdir(readOnlyDir, 0o555))
		t.Cleanup(func() {
			assert.NoError(t, os.Chmod(readOnlyDir, 0o755))
		})

		sut := Wiper{
			WipeOut: []string{"file.txt"},
		}

		errChan := make(chan error, 10)
		sut.handleFile(readOnlyDir, filepath.Join(readOnlyDir, ".Trash"), "file.txt", errChan)

		close(errChan)
		errs := make([]error, 0)
		for err := range errChan {
			if err != nil {
				errs = append(errs, err)
			}
		}
		assert.NotEmpty(t, errs, "should report error when file deletion fails")
	})
}

func TestInitTrashEdgeCases(t *testing.T) {
	t.Run("trash path permissions", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		sut := Wiper{
			UseTrash: true,
		}

		result := initTrash(&sut)
		trashPath := filepath.Join(testHome, ".Trash")

		assert.Equal(t, trashPath, result)
		info, err := os.Stat(trashPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})
}
