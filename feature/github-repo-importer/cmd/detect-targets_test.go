package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
	}{
		{
			name:     "yaml file in repos",
			filePath: "repos/my-repo.yaml",
			baseDir:  "repos",
			expected: "my-repo",
		},
		{
			name:     "yml file in repos",
			filePath: "repos/another-repo.yml",
			baseDir:  "repos",
			expected: "another-repo",
		},
		{
			name:     "file in importer_tmp_dir",
			filePath: "importer_tmp_dir/imported-repo.yaml",
			baseDir:  "importer_tmp_dir",
			expected: "imported-repo",
		},
		{
			name:     "nested path",
			filePath: "repos/subdir/repo.yaml",
			baseDir:  "repos",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoName(tt.filePath, tt.baseDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCriticalFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "terraform file",
			filePath: "main.tf",
			expected: true,
		},
		{
			name:     "terraform file in subdirectory",
			filePath: "feature/github-repo-provisioning/main.tf",
			expected: true,
		},
		{
			name:     "workflow file",
			filePath: ".github/workflows/tf-plan.yaml",
			expected: true,
		},
		{
			name:     "action file",
			filePath: ".github/actions/graformer/action.yaml",
			expected: true,
		},
		{
			name:     "backend file",
			filePath: "backend.tf.hcp",
			expected: true,
		},
		{
			name:     "markdown file",
			filePath: "README.md",
			expected: false,
		},
		{
			name:     "yaml config file",
			filePath: "repos/my-repo.yaml",
			expected: false,
		},
		{
			name:     "gitignore",
			filePath: ".gitignore",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCriticalFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzeChanges(t *testing.T) {
	tests := []struct {
		name            string
		changedFiles    []string
		reposDir        string
		importerDir     string
		expectedStrategy string
		expectedReason   string
		expectedTargets  int
	}{
		{
			name:            "no changes",
			changedFiles:    []string{},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyNone,
			expectedReason:   "No changes detected",
			expectedTargets:  0,
		},
		{
			name: "only repo changes",
			changedFiles: []string{
				"repos/repo1.yaml",
				"repos/repo2.yaml",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyTargets,
			expectedReason:   "Only repository configs changed (2 repos)",
			expectedTargets:  2,
		},
		{
			name: "only non-critical file changes",
			changedFiles: []string{
				"README.md",
				".gitignore",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyFull,
			expectedReason:   "No repository configs changed, but other files modified",
			expectedTargets:  0,
		},
		{
			name: "repo changes with markdown",
			changedFiles: []string{
				"repos/repo1.yaml",
				"README.md",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyTargets,
			expectedReason:   "Repository configs changed (1 repos) with only non-critical files",
			expectedTargets:  1,
		},
		{
			name: "repo changes with critical files",
			changedFiles: []string{
				"repos/repo1.yaml",
				"main.tf",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyFull,
			expectedReason:   "Repository configs and critical files changed (main.tf)",
			expectedTargets:  1,
		},
		{
			name: "importer directory changes",
			changedFiles: []string{
				"importer_tmp_dir/imported-repo.yaml",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyTargets,
			expectedReason:   "Only repository configs changed (1 repos)",
			expectedTargets:  1,
		},
		{
			name: "mixed repo and importer changes",
			changedFiles: []string{
				"repos/repo1.yaml",
				"importer_tmp_dir/imported-repo.yaml",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyTargets,
			expectedReason:   "Only repository configs changed (2 repos)",
			expectedTargets:  2,
		},
		{
			name: "workflow change",
			changedFiles: []string{
				".github/workflows/tf-plan.yaml",
			},
			reposDir:        "repos",
			importerDir:     "importer_tmp_dir",
			expectedStrategy: TargetingStrategyFull,
			expectedReason:   "No repository configs changed, but other files modified",
			expectedTargets:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzeChanges(tt.changedFiles, tt.reposDir, tt.importerDir)
			assert.Equal(t, tt.expectedStrategy, result.Strategy, "Strategy mismatch")
			assert.Contains(t, result.Reason, tt.expectedReason, "Reason mismatch")
			assert.Equal(t, tt.expectedTargets, len(result.Targets), "Targets count mismatch")
		})
	}
}

func TestGenerateTerraformTargets(t *testing.T) {
	tests := []struct {
		name     string
		result   TargetingResult
		expected string
	}{
		{
			name: "full plan strategy",
			result: TargetingResult{
				Strategy: TargetingStrategyFull,
				Targets:  []string{"repo1"},
			},
			expected: "",
		},
		{
			name: "none strategy",
			result: TargetingResult{
				Strategy: TargetingStrategyNone,
				Targets:  []string{},
			},
			expected: "",
		},
		{
			name: "targets strategy with one repo",
			result: TargetingResult{
				Strategy: TargetingStrategyTargets,
				Targets:  []string{"repo1"},
			},
			expected: "-target=module.repository[\"repo1\"]",
		},
		{
			name: "targets strategy with multiple repos",
			result: TargetingResult{
				Strategy: TargetingStrategyTargets,
				Targets:  []string{"repo1", "repo2"},
			},
			expected: "-target=module.repository[\"repo1\"] -target=module.repository[\"repo2\"]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTerraformTargets(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}
