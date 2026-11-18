package github

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Defaults holds the default value of every scalar repository property that the
// importer is allowed to strip. It mirrors the literal defaults set in the
// Terraform root module, loaded from the central defaults.yaml so the importer
// and Terraform resolve omitted values against the same source. vulnerability_alerts
// (visibility-dependent) and archive_on_destroy (a Terraform-only attribute never
// imported) are intentionally absent: those are never stripped.
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

	AllowUpdateBranch        *bool   `yaml:"allow_update_branch"`
	HasDiscussions           *bool   `yaml:"has_discussions"`
	WebCommitSignoffRequired *bool   `yaml:"web_commit_signoff_required"`
	SquashMergeCommitTitle   *string `yaml:"squash_merge_commit_title"`
	SquashMergeCommitMessage *string `yaml:"squash_merge_commit_message"`
	MergeCommitTitle         *string `yaml:"merge_commit_title"`
	MergeCommitMessage       *string `yaml:"merge_commit_message"`
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

// Apply nils out any scalar on repo whose value equals the default, so the
// omitempty YAML tags drop it on marshal. The always-present fields
// (description, visibility, default_branch), the null-default fields, and all
// nested/list blocks are left untouched. auto_init has no counterpart on
// Repository (it is a write-only creation parameter never read from GitHub), so
// it is not stripped here even though it is part of Defaults.
func (d *Defaults) Apply(repo *Repository) {
	stripBool(&repo.AllowMergeCommit, d.AllowMergeCommit)
	stripBool(&repo.AllowRebaseMerge, d.AllowRebaseMerge)
	stripBool(&repo.AllowSquashMerge, d.AllowSquashMerge)
	stripBool(&repo.AllowAutoMerge, d.AllowAutoMerge)
	stripBool(&repo.DeleteBranchOnMerge, d.DeleteBranchOnMerge)
	stripBool(&repo.HasIssues, d.HasIssues)
	stripBool(&repo.HasProjects, d.HasProjects)
	stripBool(&repo.HasWiki, d.HasWiki)
	stripBool(&repo.IsTemplate, d.IsTemplate)
	stripBool(&repo.Archived, d.Archived)
	stripString(&repo.HomepageURL, d.HomepageURL)
	stripString(&repo.GitignoreTemplate, d.GitignoreTemplate)
	stripString(&repo.LicenseTemplate, d.LicenseTemplate)
	stripBool(&repo.AllowUpdateBranch, d.AllowUpdateBranch)
	stripBool(&repo.HasDiscussions, d.HasDiscussions)
	stripBool(&repo.WebCommitSignoffRequired, d.WebCommitSignoffRequired)
	stripString(&repo.SquashMergeCommitTitle, d.SquashMergeCommitTitle)
	stripString(&repo.SquashMergeCommitMessage, d.SquashMergeCommitMessage)
	stripString(&repo.MergeCommitTitle, d.MergeCommitTitle)
	stripString(&repo.MergeCommitMessage, d.MergeCommitMessage)
}

func stripBool(field **bool, def *bool) {
	if *field != nil && def != nil && **field == *def {
		*field = nil
	}
}

func stripString(field **string, def *string) {
	if *field != nil && def != nil && **field == *def {
		*field = nil
	}
}
