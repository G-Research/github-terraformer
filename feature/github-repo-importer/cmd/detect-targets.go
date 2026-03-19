package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	TargetingStrategyFull    = "full"
	TargetingStrategyTargets = "targets"
	TargetingStrategyNone    = "none"
)

type TargetingResult struct {
	Strategy      string   // "full", "targets", or "none"
	Targets       []string // List of repository names to target
	Reason        string   // Human-readable reason for the decision
	ChangedRepos  int      // Number of changed repos
	ChangedOther  int      // Number of changed non-config files
	CriticalFiles []string // Critical files that triggered full plan
}

var detectTargetsCmd = &cobra.Command{
	Use:   "detect-targets",
	Short: "Detect changed repositories and determine terraform targeting strategy",
	Long: `Analyzes git diff to determine which repositories changed and whether to use
targeted terraform plan or full plan. Outputs targeting strategy and targets to a file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseRef, _ := cmd.Flags().GetString("base-ref")
		headRef, _ := cmd.Flags().GetString("head-ref")
		reposDir, _ := cmd.Flags().GetString("repos-dir")
		importerDir, _ := cmd.Flags().GetString("importer-dir")
		outputFile, _ := cmd.Flags().GetString("output-file")
		verbose, _ := cmd.Flags().GetBool("verbose")
		prNumber, _ := cmd.Flags().GetString("pr-number")

		// If PR number is provided, try to get changed files from PR
		var changedFiles []string
		var err error
		if prNumber != "" {
			changedFiles, err = getChangedFilesFromPR(prNumber)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Could not get files from PR %s: %v\n", prNumber, err)
					fmt.Fprintf(os.Stderr, "Falling back to git diff\n")
				}
				changedFiles, err = getChangedFilesFromGit(baseRef, headRef)
				if err != nil {
					return fmt.Errorf("failed to get changed files: %w", err)
				}
			}
		} else {
			changedFiles, err = getChangedFilesFromGit(baseRef, headRef)
			if err != nil {
				return fmt.Errorf("failed to get changed files: %w", err)
			}
		}

		result := analyzeChanges(changedFiles, reposDir, importerDir)

		if verbose {
			fmt.Fprintf(os.Stderr, "Targeting Strategy: %s\n", result.Strategy)
			fmt.Fprintf(os.Stderr, "Reason: %s\n", result.Reason)
			fmt.Fprintf(os.Stderr, "Changed repos: %d\n", result.ChangedRepos)
			fmt.Fprintf(os.Stderr, "Changed other files: %d\n", result.ChangedOther)
			if len(result.CriticalFiles) > 0 {
				fmt.Fprintf(os.Stderr, "Critical files: %v\n", result.CriticalFiles)
			}
		}

		// Write output to file or stdout
		output := generateTerraformTargets(result)
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "Targets written to: %s\n", outputFile)
			}
		} else {
			fmt.Print(output)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(detectTargetsCmd)
	detectTargetsCmd.Flags().String("base-ref", "origin/main", "Base git ref to compare against")
	detectTargetsCmd.Flags().String("head-ref", "HEAD", "Head git ref to compare")
	detectTargetsCmd.Flags().String("repos-dir", "repos", "Directory containing repository configs")
	detectTargetsCmd.Flags().String("importer-dir", "importer_tmp_dir", "Directory containing imported repository configs")
	detectTargetsCmd.Flags().String("output-file", "", "Output file to write targets (if empty, writes to stdout)")
	detectTargetsCmd.Flags().String("pr-number", "", "Pull request number to get changed files from (uses gh cli)")
	detectTargetsCmd.Flags().Bool("verbose", false, "Enable verbose output to stderr")
}

// getChangedFilesFromPR uses gh cli to get changed files from a PR
func getChangedFilesFromPR(prNumber string) ([]string, error) {
	cmd := exec.Command("gh", "pr", "view", prNumber, "--json", "files", "--jq", ".files.[].path")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh cli failed: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}
	return result, nil
}

// getChangedFilesFromGit uses git diff to get changed files
func getChangedFilesFromGit(baseRef, headRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef, headRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}
	return result, nil
}

// analyzeChanges analyzes the changed files and determines targeting strategy
func analyzeChanges(changedFiles []string, reposDir, importerDir string) TargetingResult {
	result := TargetingResult{
		Targets: []string{},
	}

	repoChanges := make(map[string]bool)
	var otherFiles []string
	var criticalFiles []string

	for _, file := range changedFiles {
		// Check if file is in repos directory
		if strings.HasPrefix(file, reposDir+"/") {
			repoName := extractRepoName(file, reposDir)
			if repoName != "" {
				repoChanges[repoName] = true
			}
			continue
		}

		// Check if file is in importer directory
		if strings.HasPrefix(file, importerDir+"/") {
			repoName := extractRepoName(file, importerDir)
			if repoName != "" {
				repoChanges[repoName] = true
			}
			continue
		}

		// File is not a repo config
		otherFiles = append(otherFiles, file)

		// Check if it's a critical file
		if isCriticalFile(file) {
			criticalFiles = append(criticalFiles, file)
		}
	}

	result.ChangedRepos = len(repoChanges)
	result.ChangedOther = len(otherFiles)
	result.CriticalFiles = criticalFiles

	// Convert map to slice
	for repo := range repoChanges {
		result.Targets = append(result.Targets, repo)
	}

	// Decision logic based on cases
	if result.ChangedRepos == 0 && result.ChangedOther == 0 {
		// No changes at all
		result.Strategy = TargetingStrategyNone
		result.Reason = "No changes detected"
		return result
	}

	if result.ChangedRepos == 0 && result.ChangedOther > 0 {
		// Case 1: No repo changes, but other files changed
		result.Strategy = TargetingStrategyFull
		result.Reason = "No repository configs changed, but other files modified"
		return result
	}

	if result.ChangedRepos > 0 && result.ChangedOther == 0 {
		// Case 2: Only repo changes, no other files
		result.Strategy = TargetingStrategyTargets
		result.Reason = fmt.Sprintf("Only repository configs changed (%d repos)", result.ChangedRepos)
		return result
	}

	// Case 3: Both repo changes and other files changed
	if len(criticalFiles) > 0 {
		result.Strategy = TargetingStrategyFull
		result.Reason = fmt.Sprintf("Repository configs and critical files changed (%s)", strings.Join(criticalFiles, ", "))
		return result
	}

	// Only safe files changed alongside repo configs
	result.Strategy = TargetingStrategyTargets
	result.Reason = fmt.Sprintf("Repository configs changed (%d repos) with only non-critical files", result.ChangedRepos)
	return result
}

// extractRepoName extracts repository name from file path
func extractRepoName(filePath, baseDir string) string {
	// Remove base directory prefix
	relative := strings.TrimPrefix(filePath, baseDir+"/")

	// Get the filename
	filename := filepath.Base(relative)

	// Remove extension (.yaml or .yml)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	return name
}

// isCriticalFile checks if a file is critical and should trigger full plan
func isCriticalFile(filePath string) bool {
	criticalPatterns := []string{
		"*.tf",
		".github/workflows/",
		".github/actions/",
		"backend.tf.*",
		"*.tf.hcp",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(pattern, "*") {
			// Simple glob matching for extensions
			ext := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(filePath, ext) {
				return true
			}
		} else {
			// Prefix matching
			if strings.HasPrefix(filePath, pattern) {
				return true
			}
		}
	}

	return false
}

// generateTerraformTargets generates the terraform target flags
func generateTerraformTargets(result TargetingResult) string {
	if result.Strategy == TargetingStrategyFull || result.Strategy == TargetingStrategyNone {
		// Return empty string for full plan or no changes
		return ""
	}

	// Generate target flags
	var targets []string
	for _, repo := range result.Targets {
		targets = append(targets, fmt.Sprintf("-target=module.repository[\"%s\"]", repo))
	}

	return strings.Join(targets, " ")
}
