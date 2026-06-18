package wiper

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigFilename(t *testing.T) {
	t.Run("green case - config file without extension exists", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config")
		configFile, err := os.Create(configPath)
		require.NoError(t, err)
		require.NoError(t, configFile.Close())

		result := getConfigFilename(configPath)
		assert.Equal(t, configPath, result)
	})

	t.Run("green case - config.yaml exists", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config")
		configFile, err := os.Create(configPath + ".yaml")
		require.NoError(t, err)
		require.NoError(t, configFile.Close())

		result := getConfigFilename(configPath)
		assert.Equal(t, configPath+".yaml", result)
	})

	t.Run("green case - config.yml exists", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config")
		configFile, err := os.Create(configPath + ".yml")
		require.NoError(t, err)
		require.NoError(t, configFile.Close())

		result := getConfigFilename(configPath)
		assert.Equal(t, configPath+".yml", result)
	})

	t.Run("red case - no config file exists", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "nonexistent")

		result := getConfigFilename(configPath)
		assert.Equal(t, "", result)
	})

	t.Run("precedence - config without extension takes precedence", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config")

		// Create multiple config files
		configNoExt, err := os.Create(configPath)
		require.NoError(t, err)
		require.NoError(t, configNoExt.Close())

		configYAML, err := os.Create(configPath + ".yaml")
		require.NoError(t, err)
		require.NoError(t, configYAML.Close())

		result := getConfigFilename(configPath)
		assert.Equal(t, configPath, result)
	})

	t.Run("precedence - yaml takes precedence over yml", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config")

		configYAML, err := os.Create(configPath + ".yaml")
		require.NoError(t, err)
		require.NoError(t, configYAML.Close())

		configYML, err := os.Create(configPath + ".yml")
		require.NoError(t, err)
		require.NoError(t, configYML.Close())

		result := getConfigFilename(configPath)
		assert.Equal(t, configPath+".yaml", result)
	})
}

func TestInitConfigNoFile(t *testing.T) {
	t.Run("green case - InitConfig with no config file", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Equal(t, "", wiper.BaseDir) // Should have defaults from struct
	})

	t.Run("CfgFile environment variable set", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configPath := filepath.Join(testHome, "custom_config.yaml")
		configContent := `
wipe_out:
  - "*.orig"
base_dir: "/tmp"
use_trash: false
`
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = configPath
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.NotEmpty(t, wiper.WipeOut)
	})
}

func TestInitConfigWithYAML(t *testing.T) {
	t.Run("green case - plain YAML config loads correctly", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `
wipe_out:
  - "*.orig"
  - "*.bak"
base_dir: "/tmp"
use_trash: false
exclude_file:
  - "important.txt"
`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Contains(t, wiper.WipeOut, "*.orig")
		assert.Contains(t, wiper.WipeOut, "*.bak")
		assert.Contains(t, wiper.ExcludeFile, "important.txt")
	})

	t.Run("malformed YAML handled gracefully", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `
wipe_out: [
  - "*.orig"
`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = ""
		viper.Reset()

		// InitConfig should handle malformed YAML without panicking
		assert.NotPanics(t, func() {
			InitConfig()
		})
	})
}

func TestInitConfigViaprConfigPath(t *testing.T) {
	t.Run("green case - config with wipe_out_dirs", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `
wipe_out_dirs:
  - "build"
  - "dist"
  - ".cache"
wipe_out_pattern_dirs:
  - "^\\..+"
exclude_dir:
  - "node_modules"
`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Contains(t, wiper.WipeOutDirs, "build")
		assert.Contains(t, wiper.WipeOutDirs, "dist")
		assert.Contains(t, wiper.ExcludeDir, "node_modules")
	})
}

func TestInitConfigWithoutSOPS(t *testing.T) {
	t.Run("green case - config file without SOPS metadata", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `wipe_out:
  - "*.orig"`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Contains(t, wiper.WipeOut, "*.orig")
	})
}

func TestReadPlainConfigFile(t *testing.T) {
	t.Run("green case - plain config file read successfully", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `
base_dir: "/tmp"
wipe_out:
  - "test.orig"
`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		viper.Reset()
		viper.AddConfigPath(filepath.Dir(configPath))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")

		// Should not panic
		assert.NotPanics(t, func() {
			readPlainConfigFile(configPath)
		})
	})

	t.Run("red case - config file read fails gracefully", func(t *testing.T) {
		viper.Reset()

		// Should not panic even if config doesn't exist
		assert.NotPanics(t, func() {
			readPlainConfigFile("/nonexistent/path/config.yaml")
		})
	})
}

func TestWiperUnmarshalFromViper(t *testing.T) {
	t.Run("green case - Wiper fields populated from viper config", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		configDir := path.Join(testHome, ".config", "wiper")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		configContent := `
wipe_out:
  - "*.orig"
  - "*.bak"
wipe_out_pattern:
  - ".*\\.tmp$"
wipe_out_dirs:
  - "temp"
  - "cache"
wipe_out_pattern_dirs:
  - "^\\.[^/]+$"
exclude_file:
  - "important.txt"
exclude_dir:
  - "node_modules"
base_dir: "/home/test"
use_trash: true
`
		configPath := path.Join(configDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Equal(t, 2, len(wiper.WipeOut))
		assert.Equal(t, 1, len(wiper.WipeOutPattern))
		assert.Equal(t, 2, len(wiper.WipeOutDirs))
		assert.Equal(t, 1, len(wiper.WipeOutPatternDirs))
		assert.Equal(t, 1, len(wiper.ExcludeFile))
		assert.Equal(t, 1, len(wiper.ExcludeDir))
		assert.Equal(t, "/home/test", wiper.BaseDir)
		assert.True(t, wiper.UseTrash)
	})

	t.Run("empty config creates empty Wiper", func(t *testing.T) {
		testHome := t.TempDir()
		t.Setenv("HOME", testHome)

		CfgFile = ""
		viper.Reset()

		InitConfig()

		assert.NotNil(t, wiper)
		assert.Empty(t, wiper.WipeOut)
		assert.Empty(t, wiper.BaseDir)
		assert.False(t, wiper.UseTrash)
	})
}
