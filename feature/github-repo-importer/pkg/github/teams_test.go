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
