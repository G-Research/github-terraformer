package github

import (
	"errors"
)

type Config struct {
	IsPublic      *bool             `yaml:"is_public,omitempty"`
	IgnoredRepos  []string          `yaml:"ignored_repos,omitempty"`
	SelectedRepos []string          `yaml:"selected_repos,omitempty"`
	PageSize      *int              `yaml:"page_size,omitempty"`
	Features      map[string]bool   `yaml:",inline"`
}

// IsFeatureEnabled checks if a feature flag is enabled (default: false)
func (c *Config) IsFeatureEnabled(featureName string) bool {
	if c.Features == nil {
		return false
	}
	return c.Features[featureName]
}

func (c *Config) Validate() error {
	if len(c.IgnoredRepos) > 0 && len(c.SelectedRepos) > 0 {
		return errors.New("only one list of ignored_repos or selected_repos must be provided")
	}
	return nil
}
