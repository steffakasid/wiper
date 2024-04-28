package wiper

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWipeFiles(t *testing.T) {

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

		err = sut.WipeFiles("")
		require.NoError(t, err)
		assert.NoFileExists(t, fileToDelete.Name())
	})

	t.Run("Success with pattern", func(t *testing.T) {
		testdir := t.TempDir()
		pattern := `.*\.orig`
		fileToDelete, err := os.CreateTemp(testdir, pattern)
		require.NoError(t, err)
		fileToDelete.Close()
		require.NoError(t, err)

		sut := Wiper{
			WipeOutPattern: []string{pattern},
			BaseDir:        testdir,
		}
		err = sut.WipeFiles("")
		require.NoError(t, err)
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

		err = sut.WipeFiles("")
		require.NoError(t, err)
		assert.NoFileExists(t, fileToDelete.Name())
		assert.FileExists(t, fileNotToDelete.Name())
	})

	t.Run("Exclude dir and skip files", func(t *testing.T) {
		testDir := t.TempDir()
		fileToDelete, err := os.CreateTemp(testDir, "")
		require.NoError(t, err)
		skippedDir, err := os.MkdirTemp(testDir, "")
		require.NoError(t, err)
		skippedFile, err := os.Create(path.Join(skippedDir, filepath.Base(fileToDelete.Name())))
		require.NoError(t, err)
		err = skippedFile.Close()
		require.NoError(t, err)

		sut := Wiper{
			WipeOut:    []string{filepath.Base(fileToDelete.Name())},
			ExcludeDir: []string{filepath.Base(skippedDir)},
			BaseDir:    testDir,
		}

		err = sut.WipeFiles("")
		require.NoError(t, err)
		assert.NoFileExists(t, fileToDelete.Name())
		assert.FileExists(t, skippedFile.Name())
	})
}
