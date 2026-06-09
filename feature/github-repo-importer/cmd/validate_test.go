package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newConfigDir creates a temporary config directory containing the given
// repos/*.yaml files (name -> content).
func newConfigDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	reposDir := filepath.Join(dir, "repos")
	require.NoError(t, os.MkdirAll(reposDir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(reposDir, name), []byte(content), 0o644))
	}
	return dir
}

func runValidateCmd(t *testing.T, configDir, override string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := runValidate(cmd, configDir, override)
	return out.String(), err
}

func TestValidate_ValidConfigPasses(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		"good.yaml": "default_branch: main\nvisibility: public\nhas_issues: true\n",
	})

	out, err := runValidateCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "Using built-in schema")
	assert.Contains(t, out, "ok -- ")
}

func TestValidate_SchemaViolationCitesFileAndPath(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		"bad.yaml": "default_branch: main\nvisibility: banana\nhas_issues: nope\n",
	})

	out, err := runValidateCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, filepath.Join(dir, "repos", "bad.yaml"))
	assert.Contains(t, out, "/visibility")
	assert.Contains(t, out, "/has_issues")
}

func TestValidate_MissingRequiredField(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		"missing.yaml": "visibility: public\n",
	})

	out, err := runValidateCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, "missing property 'default_branch'")
}

func TestValidate_MalformedYAMLReportedCleanly(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		"malformed.yaml": "default_branch: main\n  bad: indent\n",
	})

	out, err := runValidateCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, "invalid YAML")
	assert.Contains(t, out, filepath.Join(dir, "repos", "malformed.yaml"))
}

func TestValidate_NoFilesSkips(t *testing.T) {
	dir := newConfigDir(t, nil)

	out, err := runValidateCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "No repos/*.yaml files found")
}

func TestValidate_OrgSchemaOverrideTakesPrecedence(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		// Valid against the built-in schema, but the override only allows "internal".
		"r.yaml": "default_branch: main\nvisibility: public\n",
	})
	schemasDir := filepath.Join(dir, ".schemas")
	require.NoError(t, os.MkdirAll(schemasDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(schemasDir, "repository-config.schema.json"),
		[]byte(`{"type":"object","properties":{"visibility":{"enum":["internal"]}}}`),
		0o644,
	))

	out, err := runValidateCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, "Using schema override")
	assert.Contains(t, out, "/visibility")
}

func TestValidate_SymlinkSchemaOverrideIsIgnored(t *testing.T) {
	dir := newConfigDir(t, map[string]string{
		"r.yaml": "default_branch: main\n",
	})

	// Point .schemas at a directory outside the config dir.
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(outside, "repository-config.schema.json"),
		[]byte(`{"type":"object","required":["never_present"]}`),
		0o644,
	))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, ".schemas")))

	out, err := runValidateCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "Using built-in schema")
}
