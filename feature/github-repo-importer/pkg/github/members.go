package github

import (
	"fmt"
	"strings"
)

type MembersConfig struct {
	Members []Member `yaml:"members,omitempty"`
}

type Member struct {
	Username string           `yaml:"username" jsonschema:"required,minLength=1"`
	Role     string           `yaml:"role,omitempty" jsonschema:"enum=owner,enum=member"`
	Teams    []TeamMembership `yaml:"teams,omitempty"`
}

type TeamMembership struct {
	Name string `yaml:"name" jsonschema:"required,minLength=1"`
	Role string `yaml:"role,omitempty" jsonschema:"enum=member,enum=maintainer"`
}

func (c *MembersConfig) Validate(knownTeams []string, protectedOwners []string) []error {
	errs := c.ValidateEntries(knownTeams)
	return append(errs, c.ValidateProtectedOwners(protectedOwners)...)
}

// ValidateEntries checks the rules that hold within a single members file.
func (c *MembersConfig) ValidateEntries(knownTeams []string) []error {
	var errs []error

	teamSet := make(map[string]struct{}, len(knownTeams))
	teamsByFold := make(map[string]string, len(knownTeams))
	for _, t := range knownTeams {
		teamSet[t] = struct{}{}
		teamsByFold[strings.ToLower(t)] = t
	}

	seen := make(map[string]string, len(c.Members))
	for _, member := range c.Members {
		key := strings.ToLower(member.Username)
		first, exists := seen[key]
		switch {
		case !exists:
			seen[key] = member.Username
		case first == member.Username:
			errs = append(errs, fmt.Errorf("member %q is defined more than once in members.yaml", member.Username))
		default:
			errs = append(errs, fmt.Errorf("member %q collides with %q in members.yaml: GitHub logins are case-insensitive, so these are the same account", member.Username, first))
		}

		memberTeams := make(map[string]struct{}, len(member.Teams))
		for _, team := range member.Teams {
			teamKey := strings.ToLower(team.Name)
			if _, exists := memberTeams[teamKey]; exists {
				errs = append(errs, fmt.Errorf("member %q lists team %q more than once", member.Username, team.Name))
			}
			memberTeams[teamKey] = struct{}{}

			if _, ok := teamSet[team.Name]; ok {
				continue
			}
			if actual, defined := teamsByFold[teamKey]; defined {
				errs = append(errs, fmt.Errorf("member %q references team %q but teams.yaml defines it as %q: team references must match exactly", member.Username, team.Name, actual))
				continue
			}
			errs = append(errs, fmt.Errorf("member %q references team %q which is not defined in teams.yaml", member.Username, team.Name))
		}
	}

	return errs
}

// ValidateProtectedOwners checks that every protected owner is present and still an owner.
func (c *MembersConfig) ValidateProtectedOwners(protectedOwners []string) []error {
	var errs []error

	byUsername := make(map[string]Member, len(c.Members))
	for _, member := range c.Members {
		byUsername[strings.ToLower(member.Username)] = member
	}

	for _, owner := range protectedOwners {
		member, present := byUsername[strings.ToLower(owner)]
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
