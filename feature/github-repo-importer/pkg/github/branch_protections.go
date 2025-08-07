package github

import (
	"github.com/shurcooL/githubv4"
)

type BranchProtectionV4 struct {
	Pattern                       string                      `yaml:"pattern"`
	AllowsDeletions               *bool                       `yaml:"allows_deletions,omitempty"`
	AllowsForcePushes             *bool                       `yaml:"allows_force_pushes,omitempty"`
	ForcePushAllowances           []string                    `yaml:"force_push_bypassers,omitempty"`
	AllowsCreations               *bool                       `yaml:"allows_creations,omitempty"`
	BlocksCreations               *bool                       `yaml:"blocks_creations,omitempty"`
	EnforceAdmins                 *bool                       `yaml:"enforce_admins,omitempty"`
	PushRestrictions              []string                    `yaml:"push_restrictions,omitempty"`
	RequireConversationResolution *bool                       `yaml:"require_conversation_resolution,omitempty"`
	RequireSignedCommits          *bool                       `yaml:"require_signed_commits,omitempty"`
	RequiredLinearHistory         *bool                       `yaml:"required_linear_history,omitempty"`
	RequiredPullRequestReviews    *RequiredPullRequestReviews `yaml:"required_pull_request_reviews,omitempty"`
	RequiredStatusChecks          *RequiredStatusChecksV4     `yaml:"required_status_checks,omitempty"`
	RestrictsPushes               *bool                       `yaml:"restricts_pushes,omitempty"`
	LockBranch                    *bool                       `yaml:"lock_branch,omitempty"`
}

type RequiredPullRequestReviews struct {
	RequiredApprovingReviewCount *int     `yaml:"required_approving_review_count,omitempty"`
	DismissStaleReviews          *bool    `yaml:"dismiss_stale_reviews,omitempty"`
	RequireCodeOwnerReviews      *bool    `yaml:"require_code_owner_reviews,omitempty"`
	DismissalRestrictions        []string `yaml:"dismissal_restrictions,omitempty"` // reviewDismissalAllowances
	RestrictDismissals           *bool    `yaml:"restrict_dismissals,omitempty"`
	PullRequestBypassers         []string `yaml:"pull_request_bypassers,omitempty"` // bypassPullRequestAllowances
	RequireLastPushApproval      *bool    `yaml:"require_last_push_approval,omitempty"`
}

type RequiredStatusChecksV4 struct {
	Strict   *bool    `yaml:"strict,omitempty"`
	Contexts []string `yaml:"contexts,omitempty"`
}

type BranchProtectionRulesGraphQLQuery struct {
	Repository struct {
		BranchProtectionRules struct {
			Nodes []struct {
				Pattern                        githubv4.String
				AllowsDeletions                bool
				AllowsForcePushes              bool
				BlocksCreations                bool
				IsAdminEnforced                bool
				RequiresConversationResolution bool
				RequiresCommitSignatures       bool
				RequiresLinearHistory          bool
				RequiredApprovingReviewCount   *int
				DismissesStaleReviews          bool
				RequiresCodeOwnerReviews       bool
				RestrictsReviewDismissals      bool
				RequiresStrictStatusChecks     bool
				RequiresStatusChecks           bool
				RestrictsPushes                bool
				RequireLastPushApproval        bool
				LockBranch                     bool
				RequiredStatusCheckContexts    []githubv4.String
				BypassPullRequestAllowances    AllowanceWrapper `graphql:"bypassPullRequestAllowances(first: 100)"`
				ReviewDismissalAllowances      AllowanceWrapper `graphql:"reviewDismissalAllowances(first: 100)"`
				BypassForcePushAllowances      AllowanceWrapper `graphql:"bypassForcePushAllowances(first: 100)"`
				PushAllowances                 AllowanceWrapper `graphql:"pushAllowances(first: 100)"`
			}
		} `graphql:"branchProtectionRules(first:100)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type Actor struct {
	User UserFragment `graphql:"... on User"`
	App  AppFragment  `graphql:"... on App"`
	Team TeamFragment `graphql:"... on Team"`
}

type UserFragment struct {
	Name githubv4.String `graphql:"login"`
}

type AppFragment struct {
	Name githubv4.String `graphql:"slug"`
}

type TeamFragment struct {
	Name githubv4.String `graphql:"combinedSlug"`
}

type ActorWrapper struct {
	Actor Actor
}

type AllowanceWrapper struct {
	Nodes []ActorWrapper
}
