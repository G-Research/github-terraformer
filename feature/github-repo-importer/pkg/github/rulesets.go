package github

type Ruleset struct {
	ID           int64         `yaml:"id" jsonschema:"-"`
	Enforcement  string        `yaml:"enforcement" jsonschema:"enum=disabled,enum=active,enum=evaluate"`
	Name         string        `yaml:"name"`
	Rules        *Rule         `yaml:"rules"`
	Target       string        `yaml:"target" jsonschema:"enum=branch,enum=tag"`
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
	Operator string  `yaml:"operator" jsonschema:"enum=starts_with,enum=ends_with,enum=contains,enum=regex"`
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
	RequiredDeploymentEnvironments []string `yaml:"required_deployment_environments,omitempty" json:"required_deployment_environments" jsonschema:"minItems=1,required"`
}

type RequiredStatusChecks struct {
	RequiredCheck                    []RequiredCheck `yaml:"required_check" json:"required_status_checks" jsonschema:"minItems=1,required"`
	StrictRequiredStatusChecksPolicy *bool           `yaml:"strict_required_status_checks_policy,omitempty" json:"strict_required_status_checks_policy"`
}

type RequiredCheck struct {
	Context       string `yaml:"context" json:"context"`
	IntegrationID *int   `yaml:"-" json:"integration_id"`
	Source        string `yaml:"source"`
}

type RequiredCodeScanning struct {
	RequiredCodeScanningTool []RequiredCodeScanningTool `yaml:"required_code_scanning_tool,omitempty" json:"code_scanning_tools" jsonschema:"minItems=1,required"`
}

type RequiredCodeScanningTool struct {
	AlertsThreshold         string `yaml:"alerts_threshold,omitempty" json:"alerts_threshold" jsonschema:"required,enum=none,enum=errors,enum=errors_and_warnings,enum=all"`
	SecurityAlertsThreshold string `yaml:"security_alerts_threshold,omitempty" json:"security_alerts_threshold" jsonschema:"required,enum=none,enum=critical,enum=high_or_higher,enum=medium_or_higher,enum=all"`
	Tool                    string `yaml:"tool,omitempty" json:"tool" jsonschema:"required"`
}

type BypassActor struct {
	Name       string  `yaml:"name"`
	BypassMode *string `yaml:"bypass_mode,omitempty" jsonschema:"enum=always,enum=pull_request"`
}

type Conditions struct {
	RefName RefNameCondition `yaml:"ref_name,omitempty" jsonschema:"required"`
}

type RefNameCondition struct {
	Exclude []string `yaml:"exclude,omitempty" jsonschema:"minItems=1,required"`
	Include []string `yaml:"include,omitempty" jsonschema:"minItems=1,required"`
}
