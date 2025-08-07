package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var (
	configFilePath string
	bulkImportCmd  = &cobra.Command{
		Use:   "bulk-import",
		Short: "A command that imports all repositories from a given organization",
		PreRun: func(cmd *cobra.Command, args []string) {
			github.InitializeClients()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Config file path: ", configFilePath)

			cfg, err := DecodeConfiguration(configFilePath)
			if err != nil {
				return fmt.Errorf("failed to decode configuration: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("failed to validate configuration: %w", err)
			}

			repos, err := github.ImportRepos(*cfg)
			if err != nil {
				return fmt.Errorf("failed to import repositories: %w", err)
			}

			for _, repo := range repos {
				if err := github.WriteRepositoryToYaml(repo); err != nil {
					return fmt.Errorf("failed to handle repository: %w", err)
				}
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(bulkImportCmd)
	bulkImportCmd.Flags().StringVarP(&configFilePath, "config", "c", "./import-config.yaml", "Path to the yaml config file (defaults to ./import-config.yaml)")
}

func DecodeConfiguration(configFilePath string) (*github.Config, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}(file)

	var cfg github.Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if cfg.PageSize == nil {
		ps := github.DefaultPageSize
		cfg.PageSize = &ps
	}

	return &cfg, nil
}
