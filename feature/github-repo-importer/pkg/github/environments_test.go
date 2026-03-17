package github

import (
	"testing"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
func strPtr(s string) *string { return &s }

// -------------------------------------------------------------------
// buildTeamSlugMap
// -------------------------------------------------------------------

func TestBuildTeamSlugMap(t *testing.T) {
	teams := []*github.Team{
		{ID: github.Int64(1), Slug: github.String("dev-team")},
		{ID: github.Int64(2), Slug: github.String("ops-team")},
		{ID: nil, Slug: github.String("ignored")}, // no ID – should be skipped
	}
	m := buildTeamSlugMap(teams)
	assert.Equal(t, "dev-team", m[1])
	assert.Equal(t, "ops-team", m[2])
	assert.Len(t, m, 2)
}

// -------------------------------------------------------------------
// resolveEnvironmentReviewers
// -------------------------------------------------------------------

func TestResolveEnvironmentReviewers_UsersAndTeams(t *testing.T) {
	teamSlugMap := map[int64]string{42: "platform-team"}

	reviewers := []*github.RequiredReviewer{
		{
			Type:     github.String("User"),
			Reviewer: map[string]interface{}{"login": "alice", "id": float64(100)},
		},
		{
			Type:     github.String("Team"),
			Reviewer: map[string]interface{}{"id": float64(42), "slug": "fallback-slug"},
		},
	}

	result := resolveEnvironmentReviewers(reviewers, teamSlugMap)
	require.NotNil(t, result)
	assert.Equal(t, []string{"alice"}, result.Users)
	assert.Equal(t, []string{"platform-team"}, result.Teams)
}

func TestResolveEnvironmentReviewers_FallsBackToSlugOnMap(t *testing.T) {
	// Team ID not in the map → fall back to the slug embedded in the reviewer object.
	teamSlugMap := map[int64]string{}

	reviewers := []*github.RequiredReviewer{
		{
			Type:     github.String("Team"),
			Reviewer: map[string]interface{}{"id": float64(99), "slug": "embedded-slug"},
		},
	}

	result := resolveEnvironmentReviewers(reviewers, teamSlugMap)
	require.NotNil(t, result)
	assert.Equal(t, []string{"embedded-slug"}, result.Teams)
}

func TestResolveEnvironmentReviewers_Empty(t *testing.T) {
	result := resolveEnvironmentReviewers([]*github.RequiredReviewer{}, map[int64]string{})
	assert.Nil(t, result)
}

// -------------------------------------------------------------------
// resolveEnvironment
// -------------------------------------------------------------------

func TestResolveEnvironment_NoPolicy(t *testing.T) {
	env := &github.Environment{
		Name:                  github.String("staging"),
		CanAdminsBypass:       boolPtr(true),
		DeploymentBranchPolicy: nil,
		ProtectionRules:       nil,
	}

	result := resolveEnvironment(env, "owner", "repo", map[int64]string{})

	assert.Equal(t, "staging", result.Environment)
	assert.Equal(t, boolPtr(true), result.CanAdminsBypass)
	assert.Nil(t, result.DeploymentPolicy)
	assert.Nil(t, result.Reviewers)
	assert.Nil(t, result.WaitTimer)
	assert.Nil(t, result.PreventSelfReview)
}

func TestResolveEnvironment_ProtectedBranches(t *testing.T) {
	env := &github.Environment{
		Name:            github.String("production"),
		CanAdminsBypass: boolPtr(false),
		DeploymentBranchPolicy: &github.BranchPolicy{
			ProtectedBranches:    boolPtr(true),
			CustomBranchPolicies: boolPtr(false),
		},
	}

	result := resolveEnvironment(env, "owner", "repo", map[int64]string{})

	require.NotNil(t, result.DeploymentPolicy)
	assert.Equal(t, deploymentPolicyTypeProtectedBranches, result.DeploymentPolicy.PolicyType)
	assert.Empty(t, result.DeploymentPolicy.BranchPatterns)
}

func TestResolveEnvironment_WaitTimerAndPreventSelfReview(t *testing.T) {
	env := &github.Environment{
		Name: github.String("qa"),
		ProtectionRules: []*github.ProtectionRule{
			{
				Type:      github.String(protectionRuleTypeWaitTimer),
				WaitTimer: intPtr(30),
			},
			{
				Type:              github.String(protectionRuleTypeRequiredReviewers),
				PreventSelfReview: boolPtr(true),
				Reviewers:         []*github.RequiredReviewer{},
			},
		},
	}

	result := resolveEnvironment(env, "owner", "repo", map[int64]string{})

	require.NotNil(t, result.WaitTimer)
	assert.Equal(t, 30, *result.WaitTimer)
	require.NotNil(t, result.PreventSelfReview)
	assert.True(t, *result.PreventSelfReview)
}

func TestResolveEnvironment_ReviewersFromProtectionRule(t *testing.T) {
	teamSlugMap := map[int64]string{10: "dev-team"}

	env := &github.Environment{
		Name: github.String("test"),
		ProtectionRules: []*github.ProtectionRule{
			{
				Type: github.String(protectionRuleTypeRequiredReviewers),
				Reviewers: []*github.RequiredReviewer{
					{
						Type:     github.String("User"),
						Reviewer: map[string]interface{}{"login": "bob"},
					},
					{
						Type:     github.String("Team"),
						Reviewer: map[string]interface{}{"id": float64(10), "slug": "dev-team"},
					},
				},
			},
		},
	}

	result := resolveEnvironment(env, "owner", "repo", teamSlugMap)

	require.NotNil(t, result.Reviewers)
	assert.Equal(t, []string{"bob"}, result.Reviewers.Users)
	assert.Equal(t, []string{"dev-team"}, result.Reviewers.Teams)
}

// -------------------------------------------------------------------
// extractUser / extractTeam
// -------------------------------------------------------------------

func TestExtractUser(t *testing.T) {
	raw := map[string]interface{}{"login": "carol", "id": float64(7)}
	user, err := extractUser(raw)
	require.NoError(t, err)
	assert.Equal(t, "carol", user.GetLogin())
}

func TestExtractTeam(t *testing.T) {
	raw := map[string]interface{}{"id": float64(5), "slug": "backend"}
	team, err := extractTeam(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(5), team.GetID())
	assert.Equal(t, "backend", team.GetSlug())
}
