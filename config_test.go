package nve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_MissingFile_CreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := loadConfigFrom(path)

	assert.True(t, cfg.Versioning, "default config should have versioning enabled")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "default config file should have been created")
	assert.Contains(t, string(data), "versioning: true")
}

func TestLoadConfig_ExistingFile_ParsesValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	require.NoError(t, os.WriteFile(path, []byte("versioning: false\n"), 0644))

	cfg := loadConfigFrom(path)

	assert.False(t, cfg.Versioning, "should respect config file value")
}

func TestLoadConfig_MalformedYAML_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	require.NoError(t, os.WriteFile(path, []byte(":::not yaml\n"), 0644))

	cfg := loadConfigFrom(path)

	assert.True(t, cfg.Versioning, "malformed config should fall back to defaults")
}

func TestLoadConfig_UnreadableFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	require.NoError(t, os.WriteFile(path, []byte("versioning: false\n"), 0000))

	cfg := loadConfigFrom(path)

	assert.True(t, cfg.Versioning, "unreadable config should fall back to defaults")
}
