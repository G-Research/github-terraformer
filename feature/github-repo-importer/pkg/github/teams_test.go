package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func TestTeamsConfigValidate(t *testing.T) {
	tests := []struct {
		name       string
		config     TeamsConfig
		wantErrors []string
	}{
		{
			name: "valid config with nesting and a standalone secret team",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "platform-oncall", Parent: strPtr("platform"), Visibility: TeamVisibilityVisible},
					{Name: "security-core", Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: nil,
		},
		{
			name: "parent not defined in teams.yaml",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform-oncall", Parent: strPtr("platform")},
				},
			},
			wantErrors: []string{
				`team "platform-oncall" references parent "platform" which is not defined in teams.yaml`,
			},
		},
		{
			name: "secret team used as parent",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "security-core", Visibility: TeamVisibilitySecret},
					{Name: "security-oncall", Parent: strPtr("security-core"), Visibility: TeamVisibilityVisible},
				},
			},
			wantErrors: []string{
				`team "security-oncall" has secret team "security-core" as parent: secret teams cannot be nested`,
			},
		},
		{
			name: "secret team with a parent",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "security-core", Parent: strPtr("platform"), Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: []string{
				`secret team "security-core" cannot have a parent: secret teams cannot be nested`,
			},
		},
		{
			name: "secret team without a parent is allowed",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "security-core", Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: nil,
		},
		{
			name: "multiple violations are all reported in one call",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform", Visibility: TeamVisibilityVisible},
					{Name: "ghost-child", Parent: strPtr("ghost")},
					{Name: "security-core", Visibility: TeamVisibilitySecret},
					{Name: "security-child", Parent: strPtr("security-core"), Visibility: TeamVisibilitySecret},
				},
			},
			wantErrors: []string{
				`team "ghost-child" references parent "ghost" which is not defined in teams.yaml`,
				`team "security-child" has secret team "security-core" as parent: secret teams cannot be nested`,
				`secret team "security-child" cannot have a parent: secret teams cannot be nested`,
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
