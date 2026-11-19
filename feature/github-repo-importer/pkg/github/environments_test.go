package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGitHubManagedEnvironment(t *testing.T) {
	// copilot is created and owned by the GitHub Copilot coding agent, so the
	// importer must skip it (otherwise it is reported as drift on every run).
	assert.True(t, isGitHubManagedEnvironment("copilot"))

	// Real, GCSS-managed environments must not be skipped.
	assert.False(t, isGitHubManagedEnvironment("release"))
	assert.False(t, isGitHubManagedEnvironment("github-pages"))
	assert.False(t, isGitHubManagedEnvironment(""))
}
