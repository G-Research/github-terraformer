package github

type Environment struct {
	Environment       string                `yaml:"environment"`
	WaitTimer         *int                  `yaml:"wait_timer,omitempty"`
	CanAdminsBypass   *bool                 `yaml:"can_admins_bypass,omitempty"`
	PreventSelfReview *bool                 `yaml:"prevent_self_review,omitempty"`
	Reviewers         *EnvironmentReviewers `yaml:"reviewers,omitempty"`
	DeploymentPolicy  *DeploymentPolicy     `yaml:"deployment_policy,omitempty"`
}

type EnvironmentReviewers struct {
	Teams []string `yaml:"teams,omitempty"` // Team slugs (e.g., "platform-team")
	Users []string `yaml:"users,omitempty"` // GitHub usernames (e.g., "octocat")
}

type DeploymentPolicy struct {
	PolicyType     string   `yaml:"policy_type"`               // "protected_branches" or "selected_branches_and_tags"
	BranchPatterns []string `yaml:"branch_patterns,omitempty"` // e.g., ["release/*", "main"] - only for selected_branches_and_tags
	TagPatterns    []string `yaml:"tag_patterns,omitempty"`    // e.g., ["v*"] - only for selected_branches_and_tags
}
