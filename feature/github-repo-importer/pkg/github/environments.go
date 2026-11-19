package github

// githubManagedEnvironments are environments GitHub creates and owns rather than
// the user — currently the Copilot coding agent's "copilot" runtime environment.
// The importer skips them so they are not written to config and therefore not
// reported as drift on every run.
var githubManagedEnvironments = map[string]bool{
	"copilot": true,
}

func isGitHubManagedEnvironment(name string) bool {
	return githubManagedEnvironments[name]
}

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
	PolicyType      string           `yaml:"policy_type" jsonschema:"required,enum=protected_branches,enum=selected_branches_and_tags"`
	BranchPatterns  []string         `yaml:"branch_patterns,omitempty"`
	TagPatterns     []string         `yaml:"tag_patterns,omitempty"`
	BranchPolicyIDs map[string]int64 `yaml:"branch_policy_ids,omitempty" jsonschema:"-"`
	TagPolicyIDs    map[string]int64 `yaml:"tag_policy_ids,omitempty" jsonschema:"-"`
}
