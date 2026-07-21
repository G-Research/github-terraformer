package github

import "fmt"

type TeamsConfig struct {
	Teams []Team `yaml:"teams,omitempty"`
}

type Team struct {
	Name          string  `yaml:"name" jsonschema:"required"`
	Slug          *string `yaml:"slug,omitempty" jsonschema:"description=GitHub-generated team slug captured by the importer and used as the Terraform import ID; not meant to be set or edited by hand"`
	Description   *string `yaml:"description,omitempty"`
	Visibility    string  `yaml:"visibility,omitempty" jsonschema:"enum=visible,enum=secret"`
	Notifications *bool   `yaml:"notifications,omitempty"`
}

func (c *TeamsConfig) Validate() []error {
	var errs []error

	seen := make(map[string]struct{}, len(c.Teams))
	for _, team := range c.Teams {
		if _, exists := seen[team.Name]; exists {
			errs = append(errs, fmt.Errorf("team %q is defined more than once in teams.yaml", team.Name))
		}
		seen[team.Name] = struct{}{}
	}

	return errs
}
