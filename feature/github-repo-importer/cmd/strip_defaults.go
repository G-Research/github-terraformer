package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var (
	stripDefaultsReposDir string
	stripDefaultsConfig   string
)

var stripDefaultsCmd = &cobra.Command{
	Use:   "strip-defaults",
	Short: "Remove scalar properties equal to their default from existing repo configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStripDefaults(stripDefaultsReposDir, stripDefaultsConfig)
	},
}

// runStripDefaults rewrites every repo YAML under reposDir, removing scalar keys
// whose value equals the default, using the same logic the importer applies on
// import. Nested blocks and the always-present fields are preserved.
func runStripDefaults(reposDir, defaultsPath string) error {
	defaults, err := github.LoadDefaults(defaultsPath)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return fmt.Errorf("failed to read repos dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || (!strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml")) {
			continue
		}

		path := filepath.Join(reposDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", name, err)
		}

		var repo github.Repository
		if err := yaml.Unmarshal(data, &repo); err != nil {
			return fmt.Errorf("failed to parse %q: %w", name, err)
		}

		defaults.Apply(&repo)

		out, err := yaml.Marshal(&repo)
		if err != nil {
			return fmt.Errorf("failed to marshal %q: %w", name, err)
		}

		if err := os.WriteFile(path, out, 0o644); err != nil {
			return fmt.Errorf("failed to write %q: %w", name, err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(stripDefaultsCmd)
	stripDefaultsCmd.Flags().StringVar(&stripDefaultsReposDir, "repos", "./repos", "Path to the repos directory to slim")
	stripDefaultsCmd.Flags().StringVar(&stripDefaultsConfig, "config", "./defaults.yaml", "Path to the defaults file")
}
