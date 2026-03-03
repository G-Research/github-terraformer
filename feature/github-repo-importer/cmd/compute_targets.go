package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gr-oss-devops/github-repo-importer/pkg/targets"
	"github.com/spf13/cobra"
)

var (
	configRepoPath string
	gcssRef        string
	computeTargetsCmd = &cobra.Command{
		Use:   "compute-targets",
		Short: "Compute Terraform -target flags based on changed YAML files in the config repo",
		RunE:  runComputeTargets,
	}
)

func init() {
	rootCmd.AddCommand(computeTargetsCmd)
	computeTargetsCmd.Flags().StringVar(&configRepoPath, "config-repo-path", "", "Path to the config repo checkout")
	computeTargetsCmd.Flags().StringVar(&gcssRef, "gcss-ref", "main", "The gcss ref in use; if not 'main', targeting is skipped")
	_ = computeTargetsCmd.MarkFlagRequired("config-repo-path")
}

func runComputeTargets(cmd *cobra.Command, args []string) error {
	if gcssRef != "main" {
		fmt.Fprintf(os.Stderr, "gcss-ref is %q (not main) — TF code may have changed, skipping targeting\n", gcssRef)
		return nil
	}

	fetchCmd := exec.Command("git", "-C", configRepoPath, "fetch", "origin", "main", "--depth=1", "--quiet")
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	diffCmd := exec.Command("git", "-C", configRepoPath, "diff", "origin/main", "HEAD", "--name-only", "--",
		"repos/*.yaml", "repos/*.yml",
		"importer_tmp_dir/*.yaml", "importer_tmp_dir/*.yml")
	out, err := diffCmd.Output()
	if err != nil {
		return fmt.Errorf("git diff failed: %w", err)
	}

	changedFiles := strings.Split(strings.TrimSpace(string(out)), "\n")
	repoNames := targets.RepoNamesFromChangedFiles(changedFiles)
	if len(repoNames) == 0 {
		return nil
	}

	w := cmd.OutOrStdout()
	for _, name := range repoNames {
		fmt.Fprintf(w, "-target=module.repository[\"%s\"]\n", name)
	}
	fmt.Fprintln(w, "-target=github_repository_ruleset.ruleset")
	return nil
}
