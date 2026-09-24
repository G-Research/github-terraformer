package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeamsConfigValidate(t *testing.T) {
	tests := []struct {
		name       string
		config     TeamsConfig
		wantErrors []string
	}{
		{
			name: "valid config with a standalone secret team",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "platform-oncall", Visibility: TeamVisibilityVisible},
					{Name: "security-core", Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: nil,
		},
		{
			name: "duplicate team name rejected",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "platform", Visibility: TeamVisibilityVisible},
				},
			},
			wantErrors: []string{
				`team "platform" is defined more than once in teams.yaml`,
			},
		},
		{
			name: "multiple duplicates all reported in one call",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "security-core", Visibility: TeamVisibilitySecret},
					{Name: "security-core", Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: []string{
				`team "platform" is defined more than once in teams.yaml`,
				`team "security-core" is defined more than once in teams.yaml`,
			},
		},
		{
			name: "valid one-level nesting",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "platform-oncall", Visibility: TeamVisibilityVisible, Parent: strptr("platform")},
				},
			},
			wantErrors: nil,
		},
		{
			name: "parent not defined rejected",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "child", Visibility: TeamVisibilityVisible, Parent: strptr("ghost")},
				},
			},
			wantErrors: []string{
				`team "child" has parent "ghost" which is not defined in teams.yaml`,
			},
		},
		{
			name: "multi-level nesting rejected",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "gp", Visibility: TeamVisibilityVisible},
					{Name: "mid", Visibility: TeamVisibilityVisible, Parent: strptr("gp")},
					{Name: "leaf", Visibility: TeamVisibilityVisible, Parent: strptr("mid")},
				},
			},
			wantErrors: []string{
				`team "leaf" nests under "mid", which is itself nested under "gp"; only one level of team nesting is supported`,
			},
		},
		{
			name: "self-parent rejected",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "selfie", Visibility: TeamVisibilityVisible, Parent: strptr("selfie")},
				},
			},
			wantErrors: []string{
				`team "selfie" cannot be its own parent`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate()

			var got []string
			for _, err := range errs {
				got = append(got, err.Error())
			}
			assert.Equal(t, tt.wantErrors, got)
		})
	}
}

func strptr(s string) *string { return &s }
