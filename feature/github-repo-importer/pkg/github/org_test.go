package github

import (
	"encoding/json"
	"testing"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
)

func TestOrgTeamDecode(t *testing.T) {
	payload := `[
		{"id":1,"name":"My_Team","slug":"my-team","description":"d","privacy":"closed","notification_setting":"notifications_disabled","parent":null},
		{"id":2,"name":"Sec","slug":"sec","privacy":"secret","notification_setting":"notifications_enabled","parent":{"id":1,"name":"My_Team"}}
	]`

	var teams []*orgTeam
	err := json.Unmarshal([]byte(payload), &teams)

	assert.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Equal(t, "My_Team", teams[0].GetName())
	assert.Equal(t, "my-team", teams[0].GetSlug())
	assert.Equal(t, "closed", teams[0].GetPrivacy())
	assert.Equal(t, "notifications_disabled", teams[0].NotificationSetting)
	assert.Nil(t, teams[0].Parent)
	assert.NotNil(t, teams[1].Parent)
	assert.Equal(t, "My_Team", teams[1].GetParent().GetName())
}

func TestRejectNestedTeams(t *testing.T) {
	flat := []*orgTeam{
		{Team: github.Team{Name: github.String("platform")}},
		{Team: github.Team{Name: github.String("security")}},
	}
	assert.NoError(t, rejectNestedTeams(flat))

	nested := []*orgTeam{
		{Team: github.Team{Name: github.String("platform")}},
		{Team: github.Team{Name: github.String("oncall"), Parent: &github.Team{Name: github.String("platform")}}},
	}
	err := rejectNestedTeams(nested)
	assert.EqualError(t, err, `team "oncall" has parent team "platform": nested teams are not supported, cannot import`)
}

func TestBuildTeamsConfig(t *testing.T) {
	ghTeams := []*orgTeam{
		{
			Team: github.Team{
				Name:        github.String("Z_Team"),
				Slug:        github.String("z-team"),
				Description: github.String("last alphabetically"),
				Privacy:     github.String("closed"),
			},
			NotificationSetting: NotificationsDisabled,
		},
		{
			Team: github.Team{
				Name:    github.String("secret-club"),
				Slug:    github.String("secret-club"),
				Privacy: github.String("secret"),
			},
			NotificationSetting: NotificationsEnabled,
		},
		{
			Team: github.Team{
				Name: github.String("bare"),
			},
			NotificationSetting: NotificationsEnabled,
		},
	}

	config, err := buildTeamsConfig(ghTeams)

	assert.NoError(t, err)
	assert.Len(t, config.Teams, 3)

	assert.Equal(t, "Z_Team", config.Teams[0].Name)
	assert.Equal(t, "z-team", *config.Teams[0].Slug)
	assert.Equal(t, "last alphabetically", *config.Teams[0].Description)
	assert.Equal(t, TeamVisibilityVisible, config.Teams[0].Visibility)
	assert.False(t, *config.Teams[0].Notifications)

	assert.Equal(t, "bare", config.Teams[1].Name)
	assert.Nil(t, config.Teams[1].Slug)
	assert.Nil(t, config.Teams[1].Description)
	assert.Equal(t, TeamVisibilityVisible, config.Teams[1].Visibility)
	assert.True(t, *config.Teams[1].Notifications)

	assert.Equal(t, "secret-club", config.Teams[2].Name)
	assert.Equal(t, TeamVisibilitySecret, config.Teams[2].Visibility)
	assert.True(t, *config.Teams[2].Notifications)
}

func TestBuildTeamsConfigRejectsUnknownNotificationSetting(t *testing.T) {
	tests := []struct {
		name    string
		setting string
	}{
		{name: "absent from the API response", setting: ""},
		{name: "value we do not model", setting: "notifications_something_new"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghTeams := []*orgTeam{
				{Team: github.Team{Name: github.String("platform")}, NotificationSetting: tt.setting},
			}

			config, err := buildTeamsConfig(ghTeams)

			assert.Nil(t, config)
			assert.EqualError(t, err, `team "platform" has unexpected notification_setting `+
				`"`+tt.setting+`": expected "notifications_enabled" or "notifications_disabled"`)
		})
	}
}

func TestBuildMembersConfig(t *testing.T) {
	memberLogins := []string{"zoe", "alice", "bob"}
	adminLogins := []string{"alice"}
	rosters := []teamRoster{
		{name: "platform", maintainers: []string{"alice"}, members: []string{"bob", "outside-collaborator"}},
		{name: "audit", members: []string{"bob"}},
	}

	config := buildMembersConfig(memberLogins, adminLogins, rosters)

	assert.Equal(t, []Member{
		{Username: "alice", Role: MemberRoleOwner, Teams: []TeamMembership{{Name: "platform", Role: TeamRoleMaintainer}}},
		{Username: "bob", Role: MemberRoleMember, Teams: []TeamMembership{{Name: "audit"}, {Name: "platform"}}},
		{Username: "zoe", Role: MemberRoleMember},
	}, config.Members)
}

func TestBuildMembersConfigEmptyOrg(t *testing.T) {
	config := buildMembersConfig(nil, nil, nil)
	assert.Empty(t, config.Members)
}
