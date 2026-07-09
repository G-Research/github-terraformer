package github

type Environment struct {
	Environment       string                `yaml:"environment" jsonschema:"required"`
	WaitTimer         *int                  `yaml:"wait_timer,omitempty" jsonschema:"minimum=0,maximum=43200"`
	CanAdminsBypass   *bool                 `yaml:"can_admins_bypass,omitempty"`
	PreventSelfReview *bool                 `yaml:"prevent_self_review,omitempty"`
	Reviewers         *EnvironmentReviewers `yaml:"reviewers,omitempty"`
	DeploymentPolicy  *DeploymentPolicy     `yaml:"deployment_policy,omitempty"`
}

type EnvironmentReviewers struct {
	Teams []string `yaml:"teams,omitempty"`
	Users []string `yaml:"users,omitempty"`
}

type DeploymentPolicy struct {
	PolicyType     string   `yaml:"policy_type" jsonschema:"required,enum=protected_branches,enum=selected_branches_and_tags"`
	BranchPatterns []string `yaml:"branch_patterns,omitempty"`
	TagPatterns    []string `yaml:"tag_patterns,omitempty"`
}
