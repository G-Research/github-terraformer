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

func TestApplyStripsSimpleScalars(t *testing.T) {
	d := &Defaults{
		AllowUpdateBranch:        ptrTo(false),
		HasDiscussions:           ptrTo(false),
		WebCommitSignoffRequired: ptrTo(false),
		SquashMergeCommitTitle:   ptrTo("COMMIT_OR_PR_TITLE"),
		SquashMergeCommitMessage: ptrTo("COMMIT_MESSAGES"),
		MergeCommitTitle:         ptrTo("MERGE_MESSAGE"),
		MergeCommitMessage:       ptrTo("PR_TITLE"),
	}
	repo := &Repository{
		AllowUpdateBranch:        ptrTo(false),
		HasDiscussions:           ptrTo(false),
		WebCommitSignoffRequired: ptrTo(false),
		SquashMergeCommitTitle:   ptrTo("COMMIT_OR_PR_TITLE"),
		SquashMergeCommitMessage: ptrTo("COMMIT_MESSAGES"),
		MergeCommitTitle:         ptrTo("MERGE_MESSAGE"),
		MergeCommitMessage:       ptrTo("PR_TITLE"),
	}

	d.Apply(repo)

	assert.Nil(t, repo.AllowUpdateBranch)
	assert.Nil(t, repo.HasDiscussions)
	assert.Nil(t, repo.WebCommitSignoffRequired)
	assert.Nil(t, repo.SquashMergeCommitTitle)
	assert.Nil(t, repo.SquashMergeCommitMessage)
	assert.Nil(t, repo.MergeCommitTitle)
	assert.Nil(t, repo.MergeCommitMessage)
}

func TestApplyKeepsNonDefaultMergeCommit(t *testing.T) {
	d := &Defaults{SquashMergeCommitTitle: ptrTo("COMMIT_OR_PR_TITLE")}
	repo := &Repository{SquashMergeCommitTitle: ptrTo("PR_TITLE")}

	d.Apply(repo)

	require.NotNil(t, repo.SquashMergeCommitTitle)
	assert.Equal(t, "PR_TITLE", *repo.SquashMergeCommitTitle)
}

func TestApplyLeavesVulnerabilityAlerts(t *testing.T) {
	// vulnerability_alerts is visibility-dependent, not a flat default, so it is
	// not part of Defaults and must never be stripped.
	d := &Defaults{}
	repo := &Repository{VulnerabilityAlertsEnabled: ptrTo(false)}

	d.Apply(repo)

	assert.NotNil(t, repo.VulnerabilityAlertsEnabled)
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
