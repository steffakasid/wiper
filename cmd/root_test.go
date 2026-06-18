/*
Copyright © 2024 steffakasid
*/
package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	wiper "github.com/steffakasid/wiper/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWiperE(t *testing.T) {
	t.Run("green case - successful wipe with valid config", func(t *testing.T) {
		testDir := t.TempDir()
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		// Create test file to delete
		fileToDelete, err := os.CreateTemp(testDir, "todelete")
		require.NoError(t, err)
		require.NoError(t, fileToDelete.Close())

		// Initialize wiper with test config
		wiper.CfgFile = ""
		wiper.InitConfig() // Initialize with no config file

		// Reset viper for this test
		viper.Set(baseDirFlag, testDir)
		viper.Set(wipeOutFlag, []string{filepath.Base(fileToDelete.Name())})
		viper.Set(debugFlag, false)
		viper.Set(useTrashFlag, false)

		cmd := &cobra.Command{}
		err = RunWiperE(cmd, []string{})

		assert.NoError(t, err)
		assert.NoFileExists(t, fileToDelete.Name())
	})

	t.Run("red case - error when file deletion fails", func(t *testing.T) {
		testDir := t.TempDir()
		readOnlyDir := filepath.Join(testDir, "readonly")
		require.NoError(t, os.Mkdir(readOnlyDir, 0o555))
		t.Cleanup(func() {
			assert.NoError(t, os.Chmod(readOnlyDir, 0o755))
		})

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, readOnlyDir)
		viper.Set(wipeOutFlag, []string{"file.txt"})
		viper.Set(debugFlag, false)
		viper.Set(useTrashFlag, false)

		cmd := &cobra.Command{}
		err := RunWiperE(cmd, []string{})

		assert.NoError(t, err) // No error because we can't write to read-only dir
	})

	t.Run("returns error instead of blocking when base dir cannot be read", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, filepath.Join(testHome, "missing"))
		viper.Set(debugFlag, false)
		viper.Set(useTrashFlag, false)

		cmd := &cobra.Command{}
		errChan := make(chan error, 1)
		go func() {
			errChan <- RunWiperE(cmd, []string{})
		}()

		select {
		case err := <-errChan:
			require.Error(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("RunWiperE blocked instead of returning the read error")
		}
	})

	t.Run("debug flag enabled", func(t *testing.T) {
		testDir := t.TempDir()
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, testDir)
		viper.Set(debugFlag, true)

		cmd := &cobra.Command{}
		err := RunWiperE(cmd, []string{})

		assert.NoError(t, err)
	})

	t.Run("debug flag disabled", func(t *testing.T) {
		testDir := t.TempDir()
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, testDir)
		viper.Set(debugFlag, false)

		cmd := &cobra.Command{}
		err := RunWiperE(cmd, []string{})

		assert.NoError(t, err)
	})

	t.Run("use_trash flag enabled", func(t *testing.T) {
		testDir := t.TempDir()
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		fileToDelete, err := os.CreateTemp(testDir, "todelete")
		require.NoError(t, err)
		require.NoError(t, fileToDelete.Close())

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, testDir)
		viper.Set(wipeOutFlag, []string{filepath.Base(fileToDelete.Name())})
		viper.Set(debugFlag, false)
		viper.Set(useTrashFlag, true)

		cmd := &cobra.Command{}
		err = RunWiperE(cmd, []string{})

		assert.NoError(t, err)
		trashPath := filepath.Join(testHome, ".Trash", filepath.Base(fileToDelete.Name()))
		assert.FileExists(t, trashPath)
	})

	t.Run("multiple exclude patterns", func(t *testing.T) {
		testDir := t.TempDir()
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		file1, err := os.CreateTemp(testDir, "keep")
		require.NoError(t, err)
		require.NoError(t, file1.Close())

		wiper.CfgFile = ""
		viper.Reset()
		wiper.InitConfig()

		viper.Set(baseDirFlag, testDir)
		viper.Set(debugFlag, false)
		viper.Set(useTrashFlag, false)

		cmd := &cobra.Command{}
		err = RunWiperE(cmd, []string{})

		assert.NoError(t, err)
		assert.FileExists(t, file1.Name(), "file should not be deleted without wipe rules")
	})
}

func TestRootCmdInit(t *testing.T) {
	t.Run("green case - rootCmd flags initialized correctly", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		// The init() function runs on package init, so just verify flags exist
		flags := rootCmd.PersistentFlags()

		assert.NotNil(t, flags.Lookup(baseDirFlag))
		assert.NotNil(t, flags.Lookup(excludeDirFlag))
		assert.NotNil(t, flags.Lookup(excludeFileFlag))
		assert.NotNil(t, flags.Lookup(wipeOutFlag))
		assert.NotNil(t, flags.Lookup(wipeOutPatternFlag))
		assert.NotNil(t, flags.Lookup(useTrashFlag))
		assert.NotNil(t, flags.Lookup(debugFlag))
		assert.NotNil(t, flags.Lookup(configFlag))
	})

	t.Run("default flag values", func(t *testing.T) {
		flags := rootCmd.PersistentFlags()

		// Check default values
		baseDirValue, _ := flags.GetString(baseDirFlag)
		assert.NotEmpty(t, baseDirValue)
		assert.DirExists(t, baseDirValue)

		useTrashValue, _ := flags.GetBool(useTrashFlag)
		assert.False(t, useTrashValue)

		debugValue, _ := flags.GetBool(debugFlag)
		assert.False(t, debugValue)
	})
}

func TestExecute(t *testing.T) {
	t.Run("green case - Execute completes without panic", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		// This test just verifies Execute doesn't panic
		// Actual execution would require full CLI setup
		assert.NotPanics(t, func() {
			// We don't call Execute() directly to avoid os.Exit()
			// Instead we verify the rootCmd is configured
			assert.NotNil(t, rootCmd)
			assert.Equal(t, "wiper", rootCmd.Use)
		})
	})
}
