package github

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Defaults holds the default value of every scalar repository property that the
// importer is allowed to strip. It mirrors the literal defaults set in the
// Terraform root module, loaded from the central defaults.yaml so the importer
// and Terraform resolve omitted values against the same source. Fields whose
// Terraform default is null (e.g. vulnerability_alerts, the merge-commit titles)
// are intentionally absent: those are never stripped.
type Defaults struct {
	AllowMergeCommit    *bool   `yaml:"allow_merge_commit"`
	AllowRebaseMerge    *bool   `yaml:"allow_rebase_merge"`
	AllowSquashMerge    *bool   `yaml:"allow_squash_merge"`
	AllowAutoMerge      *bool   `yaml:"allow_auto_merge"`
	DeleteBranchOnMerge *bool   `yaml:"delete_branch_on_merge"`
	HasIssues           *bool   `yaml:"has_issues"`
	HasProjects         *bool   `yaml:"has_projects"`
	HasWiki             *bool   `yaml:"has_wiki"`
	IsTemplate          *bool   `yaml:"is_template"`
	Archived            *bool   `yaml:"archived"`
	AutoInit            *bool   `yaml:"auto_init"`
	HomepageURL         *string `yaml:"homepage_url"`
	GitignoreTemplate   *string `yaml:"gitignore_template"`
	LicenseTemplate     *string `yaml:"license_template"`
}

// LoadDefaults opens and decodes the central defaults file. The file is
// required: a missing or malformed file is an error, mirroring how the module
// hard-requires it at plan time.
func LoadDefaults(path string) (*Defaults, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open defaults file: %w", err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}(file)

	var defaults Defaults
	if err := yaml.NewDecoder(file).Decode(&defaults); err != nil {
		return nil, fmt.Errorf("failed to decode defaults YAML: %w", err)
	}

	return &defaults, nil
}
