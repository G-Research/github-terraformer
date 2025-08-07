package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var importCmd = &cobra.Command{
	Use:   "import [owner/repo]",
	Short: "Import command reads all repository details and creates a configuration yaml file",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		github.InitializeClients()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		repository := args[0]

		repo, err := github.ImportRepo(repository)
		if err != nil {
			return fmt.Errorf("failed to import repo: %w", err)
		}

		if err := github.WriteRepositoryToYaml(repo); err != nil {
			return fmt.Errorf("failed to handle repository: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
