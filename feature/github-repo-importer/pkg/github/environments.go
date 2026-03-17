package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-github/v67/github"
)

const (
	protectionRuleTypeWaitTimer         = "wait_timer"
	protectionRuleTypeRequiredReviewers = "required_reviewers"

	deploymentPolicyTypeAll                      = "all"
	deploymentPolicyTypeProtectedBranches        = "protected_branches"
	deploymentPolicyTypeSelectedBranchesAndTags  = "selected_branches_and_tags"
)

// fetchEnvironments lists all deployment environments for a repository and resolves
// their configuration (reviewers, branch policies, wait timers, etc).
// Returns nil, nil on 403 (insufficient permissions).
func fetchEnvironments(owner, repo string, orgTeams []*github.Team) ([]Environment, error) {
	teamSlugMap := buildTeamSlugMap(orgTeams)

	opts := &github.EnvironmentListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var environments []Environment
	for {
		resp, httpResp, err := v3client.Repositories.ListEnvironments(context.Background(), owner, repo, opts)
		if err != nil {
			if httpResp != nil && httpResp.StatusCode == http.StatusForbidden {
				fmt.Printf("skipping environments due to insufficient permissions: %v\n", err)
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}

		for _, env := range resp.Environments {
			environments = append(environments, resolveEnvironment(env, owner, repo, teamSlugMap))
		}

		if httpResp.NextPage == 0 {
			break
		}
		opts.Page = httpResp.NextPage
	}

	return environments, nil
}

// resolveEnvironment converts a go-github Environment into our YAML-serialisable struct.
func resolveEnvironment(env *github.Environment, owner, repo string, teamSlugMap map[int64]string) Environment {
	result := Environment{
		Environment:     env.GetName(),
		CanAdminsBypass: env.CanAdminsBypass,
	}

	for _, rule := range env.ProtectionRules {
		switch rule.GetType() {
		case protectionRuleTypeWaitTimer:
			result.WaitTimer = rule.WaitTimer

		case protectionRuleTypeRequiredReviewers:
			result.PreventSelfReview = rule.PreventSelfReview
			if len(rule.Reviewers) > 0 {
				result.Reviewers = resolveEnvironmentReviewers(rule.Reviewers, teamSlugMap)
			}
		}
	}

	if env.DeploymentBranchPolicy != nil {
		switch {
		case env.DeploymentBranchPolicy.GetProtectedBranches():
			result.DeploymentPolicy = &EnvironmentDeploymentPolicy{
				PolicyType: deploymentPolicyTypeProtectedBranches,
			}
		case env.DeploymentBranchPolicy.GetCustomBranchPolicies():
			patterns := fetchDeploymentBranchPatterns(owner, repo, env.GetName())
			result.DeploymentPolicy = &EnvironmentDeploymentPolicy{
				PolicyType:     deploymentPolicyTypeSelectedBranchesAndTags,
				BranchPatterns: patterns,
			}
		}
	}

	return result
}

// resolveEnvironmentReviewers converts raw API reviewer objects to usernames/team slugs.
func resolveEnvironmentReviewers(reviewers []*github.RequiredReviewer, teamSlugMap map[int64]string) *EnvironmentReviewers {
	result := &EnvironmentReviewers{}

	for _, r := range reviewers {
		switch r.GetType() {
		case "User":
			user, err := extractUser(r.Reviewer)
			if err != nil {
				fmt.Printf("failed to extract user reviewer: %v\n", err)
				continue
			}
			if login := user.GetLogin(); login != "" {
				result.Users = append(result.Users, login)
			}

		case "Team":
			team, err := extractTeam(r.Reviewer)
			if err != nil {
				fmt.Printf("failed to extract team reviewer: %v\n", err)
				continue
			}
			// Prefer the slug from the org teams map (more reliable) but fall back to the object's slug.
			if slug, ok := teamSlugMap[team.GetID()]; ok {
				result.Teams = append(result.Teams, slug)
			} else if slug := team.GetSlug(); slug != "" {
				result.Teams = append(result.Teams, slug)
			}
		}
	}

	if len(result.Users) == 0 && len(result.Teams) == 0 {
		return nil
	}
	return result
}

// fetchDeploymentBranchPatterns retrieves the branch/tag patterns configured for an
// environment that uses custom branch policies.
func fetchDeploymentBranchPatterns(owner, repo, environmentName string) []string {
	resp, _, err := v3client.Repositories.ListDeploymentBranchPolicies(context.Background(), owner, repo, environmentName)
	if err != nil {
		fmt.Printf("failed to fetch deployment branch policies for %s/%s environment %q: %v\n", owner, repo, environmentName, err)
		return nil
	}

	var patterns []string
	for _, p := range resp.BranchPolicies {
		if name := p.GetName(); name != "" {
			patterns = append(patterns, name)
		}
	}
	return patterns
}

// buildTeamSlugMap creates an id → slug lookup from a slice of org teams.
func buildTeamSlugMap(teams []*github.Team) map[int64]string {
	m := make(map[int64]string, len(teams))
	for _, t := range teams {
		if t.ID != nil && t.Slug != nil {
			m[*t.ID] = *t.Slug
		}
	}
	return m
}

// extractUser re-marshals the opaque Reviewer interface{} into a *github.User.
func extractUser(reviewer interface{}) (*github.User, error) {
	data, err := json.Marshal(reviewer)
	if err != nil {
		return nil, fmt.Errorf("marshal reviewer: %w", err)
	}
	var user github.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal user reviewer: %w", err)
	}
	return &user, nil
}

// extractTeam re-marshals the opaque Reviewer interface{} into a *github.Team.
func extractTeam(reviewer interface{}) (*github.Team, error) {
	data, err := json.Marshal(reviewer)
	if err != nil {
		return nil, fmt.Errorf("marshal reviewer: %w", err)
	}
	var team github.Team
	if err := json.Unmarshal(data, &team); err != nil {
		return nil, fmt.Errorf("unmarshal team reviewer: %w", err)
	}
	return &team, nil
}
