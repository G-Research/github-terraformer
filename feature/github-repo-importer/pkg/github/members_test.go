package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMembersConfigValidate(t *testing.T) {
	knownTeams := []string{"platform", "security-core"}

	tests := []struct {
		name            string
		config          MembersConfig
		protectedOwners []string
		wantErrors      []string
	}{
		{
			name: "valid config with team member and maintainer roles",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleOwner, Teams: []TeamMembership{{Name: "platform", Role: TeamRoleMaintainer}}},
					{Username: "bob", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "platform", Role: TeamRoleMaintainer}, {Name: "security-core"}}},
					{Username: "carol", Role: MemberRoleMember},
				},
			},
			protectedOwners: []string{"alice"},
			wantErrors:      nil,
		},
		{
			name: "owner given a plain member role in a team is rejected",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleOwner, Teams: []TeamMembership{{Name: "platform", Role: TeamRoleMember}}},
				},
			},
			wantErrors: []string{
				`member "alice" is an organisation owner, so team "platform" cannot use role "member": GitHub reports owners as maintainers of every team they belong to, so the plan would keep proposing this change without it ever taking effect`,
			},
		},
		{
			name: "owner with an omitted team role is rejected, since it defaults to member",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleOwner, Teams: []TeamMembership{{Name: "platform"}}},
				},
			},
			wantErrors: []string{
				`member "alice" is an organisation owner, so team "platform" needs role "maintainer": the role defaults to "member" when omitted, and GitHub reports owners as maintainers of every team they belong to, so the plan would keep proposing this change without it ever taking effect`,
			},
		},
		{
			name: "a plain member may hold either team role",
			config: MembersConfig{
				Members: []Member{
					{Username: "bob", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "platform", Role: TeamRoleMember}, {Name: "security-core"}}},
				},
			},
			wantErrors: nil,
		},
		{
			name: "duplicate username rejected",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleMember},
					{Username: "alice", Role: MemberRoleMember},
				},
			},
			wantErrors: []string{
				`member "alice" is defined more than once in members.yaml`,
			},
		},
		{
			name: "team reference not defined in teams.yaml",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "ghost"}}},
				},
			},
			wantErrors: []string{
				`member "alice" references team "ghost" which is not defined in teams.yaml`,
			},
		},
		{
			name: "duplicate team within a member's teams list rejected",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "platform"}, {Name: "platform", Role: TeamRoleMaintainer}}},
				},
			},
			wantErrors: []string{
				`member "alice" lists team "platform" more than once`,
			},
		},
		{
			name: "protected owner missing from members.yaml",
			config: MembersConfig{
				Members: []Member{
					{Username: "bob", Role: MemberRoleMember},
				},
			},
			protectedOwners: []string{"alice"},
			wantErrors: []string{
				`protected owner "alice" is missing from members.yaml: protected identities cannot be removed`,
			},
		},
		{
			name: "protected owner demoted to member",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleMember},
				},
			},
			protectedOwners: []string{"alice"},
			wantErrors: []string{
				`protected owner "alice" has role "member": protected identities cannot be demoted from owner`,
			},
		},
		{
			name: "multiple violations all reported in one call",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "ghost"}}},
					{Username: "alice", Role: MemberRoleMember},
				},
			},
			protectedOwners: []string{"dave"},
			wantErrors: []string{
				`member "alice" references team "ghost" which is not defined in teams.yaml`,
				`member "alice" is defined more than once in members.yaml`,
				`protected owner "dave" is missing from members.yaml: protected identities cannot be removed`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate(knownTeams, tt.protectedOwners)

			var got []string
			for _, err := range errs {
				got = append(got, err.Error())
			}
			assert.Equal(t, tt.wantErrors, got)
		})
	}
}
