package github

import "fmt"

// TeamsConfig models the organisation/teams.yaml file in the config repo.
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

// Validate performs the cross-record checks that cannot be expressed in JSON
// Schema alone: every `parent` must refer to a team defined in the same file,
// and `secret` teams cannot take part in a hierarchy in either direction.
//
// Unlike Config.Validate, this returns []error rather than a single error:
// cmd/validate.go's runValidate already collects and reports every violation
// across repo configs in one run, and teams validation should surface all
// problems in a file the same way instead of stopping at the first one.
func (c *TeamsConfig) Validate() []error {
	var errs []error

	teamsByName := make(map[string]Team, len(c.Teams))
	for _, team := range c.Teams {
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
