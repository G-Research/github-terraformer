package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripDefaultsRewritesFiles(t *testing.T) {
	dir := t.TempDir()
	reposDir := filepath.Join(dir, "repos")
	require.NoError(t, os.MkdirAll(reposDir, 0o755))

	repoFile := filepath.Join(reposDir, "example.yaml")
	content := "description: A repo\n" +
		"visibility: private\n" +
		"default_branch: main\n" +
		"has_issues: false\n" + // equals default -> stripped
		"has_wiki: true\n" + // differs from default -> kept
		"rulesets:\n" +
		"    - id: 1\n" +
		"      enforcement: active\n" +
		"      name: protect\n" +
		"      target: branch\n"
	require.NoError(t, os.WriteFile(repoFile, []byte(content), 0o644))

	defaultsPath := filepath.Join(dir, "defaults.yaml")
	require.NoError(t, os.WriteFile(defaultsPath, []byte("has_issues: false\nhas_wiki: false\n"), 0o644))

	require.NoError(t, runStripDefaults(reposDir, defaultsPath))

	s := readFile(t, repoFile)
	assert.NotContains(t, s, "has_issues")       // stripped (was the default)
	assert.Contains(t, s, "has_wiki: true")      // kept (non-default)
	assert.Contains(t, s, "description: A repo") // always-present
	assert.Contains(t, s, "visibility: private")
	assert.Contains(t, s, "default_branch: main")
	assert.Contains(t, s, "name: protect") // ruleset block retained
}

// strip-defaults must slim top-level scalars without touching nested blocks:
// no injected ruleset `id: 0`, and byte-identical indentation for everything it
// keeps. A full-struct round-trip would reindent and inject zero-value ids.
func TestStripDefaultsPreservesNestedBlocksVerbatim(t *testing.T) {
	dir := t.TempDir()
	reposDir := filepath.Join(dir, "repos")
	require.NoError(t, os.MkdirAll(reposDir, 0o755))

	repoFile := filepath.Join(reposDir, "example.yaml")
	content := "description: A repo\n" +
		"visibility: private\n" +
		"default_branch: main\n" +
		"has_issues: false\n" + // equals default -> stripped
		"rulesets:\n" +
		"  - enforcement: active\n" +
		"    name: Protect\n" +
		"    rules:\n" +
		"      deletion: true\n" +
		"    target: branch\n"
	require.NoError(t, os.WriteFile(repoFile, []byte(content), 0o644))

	defaultsPath := filepath.Join(dir, "defaults.yaml")
	require.NoError(t, os.WriteFile(defaultsPath, []byte("has_issues: false\n"), 0o644))

	require.NoError(t, runStripDefaults(reposDir, defaultsPath))

	s := readFile(t, repoFile)
	assert.NotContains(t, s, "has_issues")          // stripped
	assert.NotContains(t, s, "id: 0")               // no zero-value id injected
	assert.Contains(t, s, "  - enforcement: active\n") // 2-space list indent preserved
	assert.Contains(t, s, "      deletion: true\n")    // deep nesting preserved byte-for-byte
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}
