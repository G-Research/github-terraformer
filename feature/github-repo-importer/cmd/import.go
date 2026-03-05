package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var (
	importConfigPath string
	importCmd        = &cobra.Command{
		Use:   "import [owner/repo]",
		Short: "Import command reads all repository details and creates a configuration yaml file",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			github.InitializeClients()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repository := args[0]

			// Load configuration with all feature flags
			cfg, err := DecodeConfiguration(importConfigPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Pass the entire config to ImportRepo - it will check feature flags internally
			repo, err := github.ImportRepo(repository, cfg)
			if err != nil {
				return fmt.Errorf("failed to import repo: %w", err)
			}

			if err := github.WriteRepositoryToYaml(repo); err != nil {
				return fmt.Errorf("failed to handle repository: %w", err)
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringVarP(&importConfigPath, "config", "c", "./import-config.yaml", "Path to the import config file (default: ./import-config.yaml)")
}
