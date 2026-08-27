package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugifyTeamName(t *testing.T) {
	tests := map[string]string{
		"Platform Team":       "platform-team",
		"Platform-Team":       "platform-team",
		"Platform.Team":       "platform-team",
		"Platform/Team":       "platform-team",
		"Platform & Team":     "platform-team",
		"  Platform Team  ":   "platform-team",
		"Platform  Team":      "platform-team",
		"platform_core":       "platform_core",
		"Ćirilica Tím":        "cirilica-tim",
		"release_engineering": "release_engineering",
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, want, slugifyTeamName(name))
		})
	}
}

func TestTeamsConfigValidateSlugCollisions(t *testing.T) {
	tests := []struct {
		name       string
		config     TeamsConfig
		wantErrors []string
	}{
		{
			name: "names differing only by separator collide on the derived slug",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "Foo Bar"},
					{Name: "Foo-Bar"},
				},
			},
			wantErrors: []string{
				`team "Foo-Bar" collides with "Foo Bar" in teams.yaml: both produce the GitHub slug "foo-bar", and GitHub appends a numeric suffix to whichever team it creates second, so the slug this configuration imports and addresses by cannot be predicted`,
			},
		},
		{
			name: "a dot collides with a space",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "Platform Team"},
					{Name: "Platform.Team"},
				},
			},
			wantErrors: []string{
				`team "Platform.Team" collides with "Platform Team" in teams.yaml: both produce the GitHub slug "platform-team", and GitHub appends a numeric suffix to whichever team it creates second, so the slug this configuration imports and addresses by cannot be predicted`,
			},
		},
		{
			name: "surrounding whitespace collides with the trimmed name",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "Platform Team"},
					{Name: "Platform Team  "},
				},
			},
			wantErrors: []string{
				`team "Platform Team  " collides with "Platform Team" in teams.yaml: both produce the GitHub slug "platform-team", and GitHub appends a numeric suffix to whichever team it creates second, so the slug this configuration imports and addresses by cannot be predicted`,
			},
		},
		{
			name: "case-only differences keep the existing message",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "Platform"},
					{Name: "platform"},
				},
			},
			wantErrors: []string{
				`team "platform" collides with "Platform" in teams.yaml: names differing only in case produce the same GitHub team`,
			},
		},
		{
			name: "underscores are preserved and do not collide with hyphens",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "platform_core"},
					{Name: "platform-core"},
				},
			},
			wantErrors: nil,
		},
		{
			name: "a blank name is rejected",
			config: TeamsConfig{
				Teams: []Team{
					{Name: "   "},
				},
			},
			wantErrors: []string{
				`team name "   " in teams.yaml is blank: GitHub rejects a team name that is empty or only whitespace`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, err := range tt.config.Validate() {
				got = append(got, err.Error())
			}
			assert.Equal(t, tt.wantErrors, got)
		})
	}
}
