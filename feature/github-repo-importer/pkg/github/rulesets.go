package github

type Ruleset struct {
	ID           int64         `yaml:"id"`
	Enforcement  string        `yaml:"enforcement"`
	Name         string        `yaml:"name"`
	Rules        *Rule         `yaml:"rules"`
	Target       string        `yaml:"target"`
	BypassActors []BypassActor `yaml:"bypass_actors,omitempty"`
	Conditions   *Conditions   `yaml:"conditions,omitempty"`
}

type Rule struct {
	BranchNamePattern         *PatternRule          `yaml:"branch_name_pattern,omitempty"`
	CommitAuthorEmailPattern  *PatternRule          `yaml:"commit_author_email_pattern,omitempty"`
	CommitMessagePattern      *PatternRule          `yaml:"commit_message_pattern,omitempty"`
	CommitterEmailPattern     *PatternRule          `yaml:"committer_email_pattern,omitempty"`
	Creation                  *bool                 `yaml:"creation,omitempty"`
	Deletion                  *bool                 `yaml:"deletion,omitempty"`
	NonFastForward            *bool                 `yaml:"non_fast_forward,omitempty"`
	PullRequest               *PullRequestRule      `yaml:"pull_request,omitempty"`
	RequiredDeployments       *RequiredDeployments  `yaml:"required_deployments,omitempty"`
	RequiredLinearHistory     *bool                 `yaml:"required_linear_history,omitempty"`
	RequiredSignatures        *bool                 `yaml:"required_signatures,omitempty,omitempty"`
	RequiredStatusChecks      *RequiredStatusChecks `yaml:"required_status_checks,omitempty"`
	TagNamePattern            *PatternRule          `yaml:"tag_name_pattern,omitempty"`
	RequiredCodeScanning      *RequiredCodeScanning `yaml:"required_code_scanning,omitempty"`
	Update                    *bool                 `yaml:"update,omitempty"`
	UpdateAllowsFetchAndMerge *bool                 `yaml:"update_allows_fetch_and_merge,omitempty"`
}

type PatternRule struct {
	Operator string  `yaml:"operator"`
	Pattern  string  `yaml:"pattern"`
	Name     *string `yaml:"name,omitempty"`
	Negate   *bool   `yaml:"negate,omitempty"`
}

type PullRequestRule struct {
	DismissStaleReviewsOnPush      *bool `yaml:"dismiss_stale_reviews_on_push,omitempty" json:"dismiss_stale_reviews_on_push"`
	RequireCodeOwnerReview         *bool `yaml:"require_code_owner_review,omitempty" json:"require_code_owner_review"`
	RequireLastPushApproval        *bool `yaml:"require_last_push_approval,omitempty" json:"require_last_push_approval"`
	RequiredApprovingReviewCount   *int  `yaml:"required_approving_review_count,omitempty" json:"required_approving_review_count"`
	RequiredReviewThreadResolution *bool `yaml:"required_review_thread_resolution,omitempty" json:"required_review_thread_resolution"`
}

type RequiredDeployments struct {
	RequiredDeploymentEnvironments []string `yaml:"required_deployment_environments,omitempty" json:"required_deployment_environments"`
}

type RequiredStatusChecks struct {
	RequiredCheck                    []RequiredCheck `yaml:"required_check" json:"required_status_checks"`
	StrictRequiredStatusChecksPolicy *bool           `yaml:"strict_required_status_checks_policy,omitempty" json:"strict_required_status_checks_policy"`
}

type RequiredCheck struct {
	Context       string `yaml:"context" json:"context"`
	IntegrationID *int   `yaml:"integration_id,omitempty" json:"integration_id"`
}

type RequiredCodeScanning struct {
	RequiredCodeScanningTool []RequiredCodeScanningTool `yaml:"required_code_scanning_tool,omitempty" json:"code_scanning_tools"`
}

type RequiredCodeScanningTool struct {
	AlertsThreshold         string `yaml:"alerts_threshold,omitempty" json:"alerts_threshold"`
	SecurityAlertsThreshold string `yaml:"security_alerts_threshold,omitempty" json:"security_alerts_threshold"`
	Tool                    string `yaml:"tool,omitempty" json:"tool"`
}

type BypassActor struct {
	Name       string  `yaml:"name"`
	BypassMode *string `yaml:"bypass_mode,omitempty"`
}

type Conditions struct {
	RefName RefNameCondition `yaml:"ref_name,omitempty"`
}

type RefNameCondition struct {
	Exclude []string `yaml:"exclude,omitempty"`
	Include []string `yaml:"include,omitempty"`
}
