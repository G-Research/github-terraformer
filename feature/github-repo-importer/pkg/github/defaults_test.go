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

// ImportRepos must surface a clean error when a selected repo fails to import,
// not panic dereferencing the nil *Repository the failing ImportRepo returns.
func TestImportReposErrorDoesNotPanicOnNilRepo(t *testing.T) {
	_, err := ImportRepos(Config{SelectedRepos: []string{"invalid-format"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid-format")
}

func ptrTo[T any](v T) *T { return &v }

func TestApplyStripsDefaults(t *testing.T) {
	d := &Defaults{HasIssues: ptrTo(false), AllowMergeCommit: ptrTo(true), HomepageURL: ptrTo("")}
	repo := &Repository{HasIssues: ptrTo(false), AllowMergeCommit: ptrTo(true), HomepageURL: ptrTo("")}

	d.Apply(repo)

	assert.Nil(t, repo.HasIssues)
	assert.Nil(t, repo.AllowMergeCommit)
	assert.Nil(t, repo.HomepageURL)
}

func TestApplyKeepsNonDefaults(t *testing.T) {
	d := &Defaults{HasIssues: ptrTo(false), HomepageURL: ptrTo("")}
	repo := &Repository{HasIssues: ptrTo(true), HomepageURL: ptrTo("https://example.com")}

	d.Apply(repo)

	require.NotNil(t, repo.HasIssues)
	assert.Equal(t, true, *repo.HasIssues)
	require.NotNil(t, repo.HomepageURL)
	assert.Equal(t, "https://example.com", *repo.HomepageURL)
}

func TestApplyKeepsAlwaysPresent(t *testing.T) {
	d := &Defaults{}
	repo := &Repository{Description: ptrTo(""), Visibility: "private", DefaultBranch: "main"}

	d.Apply(repo)

	require.NotNil(t, repo.Description)
	assert.Equal(t, "", *repo.Description)
	assert.Equal(t, "private", repo.Visibility)
	assert.Equal(t, "main", repo.DefaultBranch)
}

func TestApplyLeavesNullDefaultFields(t *testing.T) {
	d := &Defaults{}
	repo := &Repository{
		VulnerabilityAlertsEnabled: ptrTo(false),
		SquashMergeCommitTitle:     ptrTo("PR_TITLE"),
		AllowUpdateBranch:          ptrTo(false),
	}

	d.Apply(repo)

	assert.NotNil(t, repo.VulnerabilityAlertsEnabled)
	assert.NotNil(t, repo.SquashMergeCommitTitle)
	assert.NotNil(t, repo.AllowUpdateBranch)
}

func TestApplyIgnoresNestedBlocks(t *testing.T) {
	d := &Defaults{HasIssues: ptrTo(false)}
	repo := &Repository{
		HasIssues:         ptrTo(false),
		Rulesets:          []Ruleset{{Name: "protect"}},
		PushCollaborators: []string{"alice"},
	}

	d.Apply(repo)

	assert.Nil(t, repo.HasIssues)
	assert.Len(t, repo.Rulesets, 1)
	assert.Equal(t, []string{"alice"}, repo.PushCollaborators)
}
