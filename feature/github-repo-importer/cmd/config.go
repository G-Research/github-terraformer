package cmd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

// DecodeConfiguration reads and decodes the import configuration file
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
