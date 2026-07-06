package github

import "fmt"

type TeamsConfig struct {
	Teams []Team `yaml:"teams,omitempty"`
}

type Team struct {
	Name          string  `yaml:"name" jsonschema:"required"`
	Description   *string `yaml:"description,omitempty"`
	Parent        *string `yaml:"parent,omitempty"`
	Visibility    string  `yaml:"visibility,omitempty" jsonschema:"enum=visible,enum=secret"`
	Notifications *bool   `yaml:"notifications,omitempty"`
}

func (c *TeamsConfig) Validate() []error {
	var errs []error

	teamsByName := make(map[string]Team, len(c.Teams))
	for _, team := range c.Teams {
		if _, exists := teamsByName[team.Name]; exists {
			errs = append(errs, fmt.Errorf("team %q is defined more than once in teams.yaml", team.Name))
		}
		teamsByName[team.Name] = team
	}

	for _, team := range c.Teams {
		if team.Parent == nil {
			continue
		}

		parent, defined := teamsByName[*team.Parent]
		if !defined {
			errs = append(errs, fmt.Errorf("team %q references parent %q which is not defined in teams.yaml", team.Name, *team.Parent))
			continue
		}

		if parent.Visibility == TeamVisibilitySecret {
			errs = append(errs, fmt.Errorf("team %q has secret team %q as parent: secret teams cannot be nested", team.Name, *team.Parent))
		}

		if team.Visibility == TeamVisibilitySecret {
			errs = append(errs, fmt.Errorf("secret team %q cannot have a parent: secret teams cannot be nested", team.Name))
		}
	}

	return errs
}
