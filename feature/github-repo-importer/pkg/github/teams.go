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
	Parent        *string `yaml:"parent,omitempty" jsonschema:"description=Name of the parent team (must be another team defined in teams.yaml). Omit for a top-level team."`
}

func (c *TeamsConfig) Validate() []error {
	var errs []error

	seen := make(map[string]string, len(c.Teams))
	for _, team := range c.Teams {
		key := strings.ToLower(team.Name)
		first, exists := seen[key]
		switch {
		case !exists:
			seen[key] = team.Name
		case first == team.Name:
			errs = append(errs, fmt.Errorf("team %q is defined more than once in teams.yaml", team.Name))
		default:
			errs = append(errs, fmt.Errorf("team %q collides with %q in teams.yaml: names differing only in case produce the same GitHub team", team.Name, first))
		}
	}

	// Validate parent (nested team) references.
	parentOf := make(map[string]*string, len(c.Teams))
	for _, team := range c.Teams {
		parentOf[team.Name] = team.Parent
	}
	for _, team := range c.Teams {
		if team.Parent == nil {
			continue
		}
		parent := *team.Parent
		parentsParent, parentIsDefined := parentOf[parent]
		switch {
		case parent == team.Name:
			// Rule 3: a team cannot be its own parent.
			errs = append(errs, fmt.Errorf("team %q cannot be its own parent", team.Name))
		case !parentIsDefined:
			// Rule 1: parent must be a team defined in teams.yaml.
			errs = append(errs, fmt.Errorf("team %q has parent %q which is not defined in teams.yaml", team.Name, parent))
		case parentsParent != nil:
			// Rule 2: the parent must itself be top-level — only one level of nesting is supported.
			errs = append(errs, fmt.Errorf("team %q nests under %q, which is itself nested under %q; only one level of team nesting is supported", team.Name, parent, *parentsParent))
		}
	}

	return errs
}
