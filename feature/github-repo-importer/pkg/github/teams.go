package github

import (
	"fmt"
	"strings"
)

type TeamsConfig struct {
	Teams []Team `yaml:"teams,omitempty"`
}

type Team struct {
	Name          string  `yaml:"name" jsonschema:"required,minLength=1"`
	Slug          *string `yaml:"slug,omitempty" jsonschema:"description=GitHub-generated team slug captured by the importer and used as the Terraform import ID; not meant to be set or edited by hand"`
	Description   *string `yaml:"description,omitempty"`
	Visibility    string  `yaml:"visibility,omitempty" jsonschema:"enum=visible,enum=secret"`
	Notifications *bool   `yaml:"notifications,omitempty"`
}

func (c *TeamsConfig) Validate() []error {
	var errs []error

	seen := make(map[string]string, len(c.Teams))
	for _, team := range c.Teams {
		if strings.TrimSpace(team.Name) == "" {
			errs = append(errs, fmt.Errorf("team name %q in teams.yaml is blank: GitHub rejects a team name that is empty or only whitespace", team.Name))
			continue
		}

		slug := slugifyTeamName(team.Name)
		key := slug
		if key == "" {
			key = strings.ToLower(team.Name)
		}

		first, exists := seen[key]
		switch {
		case !exists:
			seen[key] = team.Name
		case first == team.Name:
			errs = append(errs, fmt.Errorf("team %q is defined more than once in teams.yaml", team.Name))
		case strings.EqualFold(first, team.Name):
			errs = append(errs, fmt.Errorf("team %q collides with %q in teams.yaml: names differing only in case produce the same GitHub team", team.Name, first))
		default:
			errs = append(errs, fmt.Errorf("team %q collides with %q in teams.yaml: both produce the GitHub slug %q, and GitHub appends a numeric suffix to whichever team it creates second, so the slug this configuration imports and addresses by cannot be predicted", team.Name, first, slug))
		}
	}

	return errs
}
