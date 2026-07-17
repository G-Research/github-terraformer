package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/go-github/v67/github"
	"gopkg.in/yaml.v3"
)

type orgTeam struct {
	github.Team
	NotificationSetting string `json:"notification_setting"`
}

type teamRoster struct {
	name        string
	maintainers []string
	members     []string
}

func ImportOrg(org string) (*TeamsConfig, *MembersConfig, error) {
	ctx := context.Background()

	ghTeams, err := listAllTeams(ctx, org)
	if err != nil {
		return nil, nil, err
	}
	if err := rejectNestedTeams(ghTeams); err != nil {
		return nil, nil, err
	}

	memberLogins, err := listMemberLogins(ctx, org, "all")
	if err != nil {
		return nil, nil, err
	}
	adminLogins, err := listMemberLogins(ctx, org, "admin")
	if err != nil {
		return nil, nil, err
	}

	rosters, err := fetchTeamRosters(ctx, org, ghTeams)
	if err != nil {
		return nil, nil, err
	}

	return buildTeamsConfig(ghTeams), buildMembersConfig(memberLogins, adminLogins, rosters), nil
}

func listAllTeams(ctx context.Context, org string) ([]*orgTeam, error) {
	var all []*orgTeam
	page := 1
	for {
		req, err := v3client.NewRequest("GET", fmt.Sprintf("orgs/%s/teams?per_page=%d&page=%d", org, DefaultPageSize, page), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build teams request: %w", err)
		}
		var teams []*orgTeam
		resp, err := v3client.Do(ctx, req, &teams)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}
		all = append(all, teams...)
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}

func rejectNestedTeams(teams []*orgTeam) error {
	for _, t := range teams {
		if t.Parent != nil {
			return fmt.Errorf("team %q has parent team %q: nested teams are not supported, cannot import", t.GetName(), t.GetParent().GetName())
		}
	}
	return nil
}

func buildTeamsConfig(ghTeams []*orgTeam) *TeamsConfig {
	teams := make([]Team, 0, len(ghTeams))
	for _, t := range ghTeams {
		team := Team{Name: t.GetName()}

		if slug := t.GetSlug(); slug != "" {
			s := slug
			team.Slug = &s
		}

		if desc := t.GetDescription(); desc != "" {
			d := desc
			team.Description = &d
		}

		if t.GetPrivacy() == "secret" {
			team.Visibility = TeamVisibilitySecret
		} else {
			team.Visibility = TeamVisibilityVisible
		}

		notifications := t.NotificationSetting != "notifications_disabled"
		team.Notifications = &notifications

		teams = append(teams, team)
	}

	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })
	return &TeamsConfig{Teams: teams}
}

func listMemberLogins(ctx context.Context, org, role string) ([]string, error) {
	var logins []string
	opts := &github.ListMembersOptions{Role: role, ListOptions: github.ListOptions{PerPage: DefaultPageSize}}
	for {
		users, resp, err := v3client.Organizations.ListMembers(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org members (role %s): %w", role, err)
		}
		for _, u := range users {
			logins = append(logins, u.GetLogin())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return logins, nil
}

func fetchTeamRosters(ctx context.Context, org string, ghTeams []*orgTeam) ([]teamRoster, error) {
	rosters := make([]teamRoster, 0, len(ghTeams))
	for _, t := range ghTeams {
		maintainers, err := listTeamMemberLogins(ctx, org, t.GetSlug(), "maintainer")
		if err != nil {
			return nil, err
		}
		members, err := listTeamMemberLogins(ctx, org, t.GetSlug(), "member")
		if err != nil {
			return nil, err
		}
		rosters = append(rosters, teamRoster{name: t.GetName(), maintainers: maintainers, members: members})
	}
	return rosters, nil
}

func listTeamMemberLogins(ctx context.Context, org, teamSlug, role string) ([]string, error) {
	var logins []string
	opts := &github.TeamListTeamMembersOptions{Role: role, ListOptions: github.ListOptions{PerPage: DefaultPageSize}}
	for {
		users, resp, err := v3client.Teams.ListTeamMembersBySlug(ctx, org, teamSlug, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s of team %q: %w", role, teamSlug, err)
		}
		for _, u := range users {
			logins = append(logins, u.GetLogin())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return logins, nil
}

func buildMembersConfig(memberLogins, adminLogins []string, rosters []teamRoster) *MembersConfig {
	adminSet := make(map[string]struct{}, len(adminLogins))
	for _, login := range adminLogins {
		adminSet[login] = struct{}{}
	}

	membersByLogin := make(map[string]*Member, len(memberLogins))
	for _, login := range memberLogins {
		role := MemberRoleMember
		if _, ok := adminSet[login]; ok {
			role = MemberRoleOwner
		}
		membersByLogin[login] = &Member{Username: login, Role: role}
	}

	for _, roster := range rosters {
		for _, login := range roster.maintainers {
			if member, ok := membersByLogin[login]; ok {
				member.Teams = append(member.Teams, TeamMembership{Name: roster.name, Role: TeamRoleMaintainer})
			}
		}
		for _, login := range roster.members {
			if member, ok := membersByLogin[login]; ok {
				member.Teams = append(member.Teams, TeamMembership{Name: roster.name})
			}
		}
	}

	members := make([]Member, 0, len(membersByLogin))
	for _, m := range membersByLogin {
		sort.Slice(m.Teams, func(i, j int) bool { return m.Teams[i].Name < m.Teams[j].Name })
		members = append(members, *m)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Username < members[j].Username })
	return &MembersConfig{Members: members}
}

func WriteOrgConfig(org string, teams *TeamsConfig, members *MembersConfig) error {
	basePath := filepath.Join("./configs", org, "organisation")
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create org config directory: %w", err)
	}

	teamsData, err := yaml.Marshal(teams)
	if err != nil {
		return fmt.Errorf("failed to marshal teams: %w", err)
	}
	if err := os.WriteFile(filepath.Join(basePath, "teams.yaml"), teamsData, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write teams.yaml: %w", err)
	}

	membersData, err := yaml.Marshal(members)
	if err != nil {
		return fmt.Errorf("failed to marshal members: %w", err)
	}
	if err := os.WriteFile(filepath.Join(basePath, "members.yaml"), membersData, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write members.yaml: %w", err)
	}

	return nil
}
