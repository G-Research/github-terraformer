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

func listAllTeams(ctx context.Context, org string) ([]*github.Team, error) {
	var all []*github.Team
	opts := &github.ListOptions{PerPage: DefaultPageSize}
	for {
		teams, resp, err := v3client.Teams.ListTeams(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}
		for _, t := range teams {
			if t.Parent != nil {
				return nil, fmt.Errorf("team %q has parent team %q: nested teams are not supported, cannot import", t.GetName(), t.GetParent().GetName())
			}
		}
		all = append(all, teams...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func getTeamNotificationSetting(ctx context.Context, org, slug string) (string, error) {
	req, err := v3client.NewRequest("GET", fmt.Sprintf("orgs/%s/teams/%s", org, slug), nil)
	if err != nil {
		return "", fmt.Errorf("failed to build team request: %w", err)
	}
	var detail struct {
		NotificationSetting string `json:"notification_setting"`
	}
	if _, err := v3client.Do(ctx, req, &detail); err != nil {
		return "", fmt.Errorf("failed to get team %q: %w", slug, err)
	}
	return detail.NotificationSetting, nil
}

func ImportOrgTeams(org string) (*TeamsConfig, error) {
	ctx := context.Background()

	ghTeams, err := listAllTeams(ctx, org)
	if err != nil {
		return nil, err
	}

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

		setting, err := getTeamNotificationSetting(ctx, org, t.GetSlug())
		if err != nil {
			return nil, err
		}
		notifications := setting != "notifications_disabled"
		team.Notifications = &notifications

		teams = append(teams, team)
	}

	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })
	return &TeamsConfig{Teams: teams}, nil
}

func listOrgMemberLogins(ctx context.Context, org string) ([]string, error) {
	var logins []string
	opts := &github.ListMembersOptions{ListOptions: github.ListOptions{PerPage: DefaultPageSize}}
	for {
		users, resp, err := v3client.Organizations.ListMembers(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org members: %w", err)
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

func addTeamMembersWithRole(ctx context.Context, org, teamSlug, teamName, apiRole, yamlRole string, membersByLogin map[string]*Member) error {
	opts := &github.TeamListTeamMembersOptions{Role: apiRole, ListOptions: github.ListOptions{PerPage: DefaultPageSize}}
	for {
		users, resp, err := v3client.Teams.ListTeamMembersBySlug(ctx, org, teamSlug, opts)
		if err != nil {
			return fmt.Errorf("failed to list %s of team %q: %w", apiRole, teamSlug, err)
		}
		for _, u := range users {
			member, ok := membersByLogin[u.GetLogin()]
			if !ok {
				continue
			}
			member.Teams = append(member.Teams, TeamMembership{Name: teamName, Role: yamlRole})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return nil
}

func ImportOrgMembers(org string) (*MembersConfig, error) {
	ctx := context.Background()

	logins, err := listOrgMemberLogins(ctx, org)
	if err != nil {
		return nil, err
	}

	membersByLogin := make(map[string]*Member, len(logins))
	for _, login := range logins {
		membership, _, err := v3client.Organizations.GetOrgMembership(ctx, login, org)
		if err != nil {
			return nil, fmt.Errorf("failed to get membership for %q: %w", login, err)
		}
		role := MemberRoleMember
		if membership.GetRole() == "admin" {
			role = MemberRoleOwner
		}
		membersByLogin[login] = &Member{Username: login, Role: role}
	}

	ghTeams, err := listAllTeams(ctx, org)
	if err != nil {
		return nil, err
	}
	for _, t := range ghTeams {
		if err := addTeamMembersWithRole(ctx, org, t.GetSlug(), t.GetName(), "maintainer", TeamRoleMaintainer, membersByLogin); err != nil {
			return nil, err
		}
		if err := addTeamMembersWithRole(ctx, org, t.GetSlug(), t.GetName(), "member", "", membersByLogin); err != nil {
			return nil, err
		}
	}

	members := make([]Member, 0, len(membersByLogin))
	for _, m := range membersByLogin {
		sort.Slice(m.Teams, func(i, j int) bool { return m.Teams[i].Name < m.Teams[j].Name })
		members = append(members, *m)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Username < members[j].Username })
	return &MembersConfig{Members: members}, nil
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
