package github

import "fmt"

type MembersConfig struct {
	Members []Member `yaml:"members,omitempty"`
}

type Member struct {
	Username string           `yaml:"username" jsonschema:"required"`
	Role     string           `yaml:"role,omitempty" jsonschema:"enum=owner,enum=member"`
	Teams    []TeamMembership `yaml:"teams,omitempty"`
}

type TeamMembership struct {
	Name string `yaml:"name" jsonschema:"required"`
	Role string `yaml:"role,omitempty" jsonschema:"enum=member,enum=maintainer"`
}

func (c *MembersConfig) Validate(knownTeams []string, protectedOwners []string) []error {
	var errs []error

	teamSet := make(map[string]struct{}, len(knownTeams))
	for _, t := range knownTeams {
		teamSet[t] = struct{}{}
	}

	seen := make(map[string]Member, len(c.Members))
	for _, member := range c.Members {
		if _, exists := seen[member.Username]; exists {
			errs = append(errs, fmt.Errorf("member %q is defined more than once in members.yaml", member.Username))
		}
		seen[member.Username] = member

		memberTeams := make(map[string]struct{}, len(member.Teams))
		for _, team := range member.Teams {
			if _, exists := memberTeams[team.Name]; exists {
				errs = append(errs, fmt.Errorf("member %q lists team %q more than once", member.Username, team.Name))
			}
			memberTeams[team.Name] = struct{}{}

			if _, ok := teamSet[team.Name]; !ok {
				errs = append(errs, fmt.Errorf("member %q references team %q which is not defined in teams.yaml", member.Username, team.Name))
			}
		}
	}

	for _, owner := range protectedOwners {
		member, present := seen[owner]
		if !present {
			errs = append(errs, fmt.Errorf("protected owner %q is missing from members.yaml: protected identities cannot be removed", owner))
			continue
		}
		if member.Role != MemberRoleOwner {
			errs = append(errs, fmt.Errorf("protected owner %q has role %q: protected identities cannot be demoted from owner", owner, member.Role))
		}
	}

	return errs
}
