package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaults.yaml")
	content := "allow_merge_commit: true\n" +
		"has_issues: false\n" +
		"auto_init: true\n" +
		"homepage_url: \"\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	d, err := LoadDefaults(path)
	require.NoError(t, err)
	assert.Equal(t, true, *d.AllowMergeCommit)
	assert.Equal(t, false, *d.HasIssues)
	assert.Equal(t, true, *d.AutoInit)
	assert.Equal(t, "", *d.HomepageURL)
}

func TestLoadDefaultsMissingFileErrors(t *testing.T) {
	_, err := LoadDefaults(filepath.Join(t.TempDir(), "nope.yaml"))
	assert.Error(t, err)
}
