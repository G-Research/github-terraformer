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
			name: "valid config",
			config: MembersConfig{
				Members: []Member{
					{Username: "alice", Role: MemberRoleOwner, Teams: []string{"platform"}},
					{Username: "bob", Role: MemberRoleMember, Teams: []string{"platform", "security-core"}},
					{Username: "carol", Role: MemberRoleMember},
				},
			},
			protectedOwners: []string{"alice"},
			wantErrors:      nil,
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
					{Username: "alice", Role: MemberRoleMember, Teams: []string{"ghost"}},
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
					{Username: "alice", Role: MemberRoleMember, Teams: []string{"platform", "platform"}},
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
					{Username: "alice", Role: MemberRoleMember, Teams: []string{"ghost"}},
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
