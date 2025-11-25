package github

type Repository struct {
	Name                       string                `yaml:"-"`
	Owner                      string                `yaml:"-"`
	Description                *string               `yaml:"description,omitempty"`
	Visibility                 string                `yaml:"visibility,omitempty" jsonschema:"enum=public,enum=private"`
	HomepageURL                *string               `yaml:"homepage_url,omitempty"`
	DefaultBranch              string                `yaml:"default_branch,omitempty" jsonschema:"required"`
	HasIssues                  *bool                 `yaml:"has_issues,omitempty"`
	HasProjects                *bool                 `yaml:"has_projects,omitempty"`
	HasWiki                    *bool                 `yaml:"has_wiki,omitempty"`
	HasDownloads               *bool                 `yaml:"has_downloads,omitempty"`
	AllowMergeCommit           *bool                 `yaml:"allow_merge_commit,omitempty"`
	AllowRebaseMerge           *bool                 `yaml:"allow_rebase_merge,omitempty"`
	AllowSquashMerge           *bool                 `yaml:"allow_squash_merge,omitempty"`
	AllowAutoMerge             *bool                 `yaml:"allow_auto_merge,omitempty"`
	AllowUpdateBranch          *bool                 `yaml:"allow_update_branch,omitempty"`
	SquashMergeCommitTitle     *string               `yaml:"squash_merge_commit_title,omitempty" jsonschema:"enum=PR_TITLE,enum=COMMIT_OR_PR_TITLE"`
	SquashMergeCommitMessage   *string               `yaml:"squash_merge_commit_message,omitempty" jsonschema:"enum=PR_BODY,enum=COMMIT_MESSAGES,enum=BLANK"`
	MergeCommitTitle           *string               `yaml:"merge_commit_title,omitempty" jsonschema:"enum=PR_TITLE,enum=MERGE_MESSAGE"`
	MergeCommitMessage         *string               `yaml:"merge_commit_message,omitempty" jsonschema:"enum=PR_BODY,enum=PR_TITLE,enum=BLANK"`
	WebCommitSignoffRequired   *bool                 `yaml:"web_commit_signoff_required,omitempty"`
	DeleteBranchOnMerge        *bool                 `yaml:"delete_branch_on_merge,omitempty"`
	IsTemplate                 *bool                 `yaml:"is_template,omitempty"`
	Archived                   *bool                 `yaml:"archived,omitempty"`
	HasDiscussions             *bool                 `yaml:"has_discussions,omitempty"`
	Topics                     []string              `yaml:"topics,omitempty"`
	PullCollaborators          []string              `yaml:"pull_collaborators,omitempty"`
	TriageCollaborators        []string              `yaml:"triage_collaborators,omitempty"`
	PushCollaborators          []string              `yaml:"push_collaborators,omitempty"`
	MaintainCollaborators      []string              `yaml:"maintain_collaborators,omitempty"`
	AdminCollaborators         []string              `yaml:"admin_collaborators,omitempty"`
	PullTeams                  []string              `yaml:"pull_teams,omitempty"`
	TriageTeams                []string              `yaml:"triage_teams,omitempty"`
	PushTeams                  []string              `yaml:"push_teams,omitempty"`
	MaintainTeams              []string              `yaml:"maintain_teams,omitempty"`
	AdminTeams                 []string              `yaml:"admin_teams,omitempty"`
	LicenseTemplate            *string               `yaml:"license_template,omitempty"`
	GitignoreTemplate          *string               `yaml:"gitignore_template,omitempty"`
	Template                   *RepositoryTemplate   `yaml:"template,omitempty"`
	Pages                      *Pages                `yaml:"pages,omitempty"`
	Rulesets                   []Ruleset             `yaml:"rulesets,omitempty"`
	VulnerabilityAlertsEnabled *bool                 `yaml:"vulnerability_alerts_enabled,omitempty"`
	BranchProtectionsV4        []*BranchProtectionV4 `yaml:"branch_protections_v4,omitempty"`
}

type RepositoryTemplate struct {
	Owner      string `yaml:"owner,omitempty" jsonschema:"required"`
	Repository string `yaml:"repository,omitempty" jsonschema:"required"`
}

type Pages struct {
	CNAME     *string `yaml:"cname,omitempty"`
	Branch    *string `yaml:"branch,omitempty" jsonschema:"required"`
	Path      *string `yaml:"path,omitempty"`
	BuildType *string `yaml:"build_type,omitempty" jsonschema:"required,enum=workflow,enum=legacy"`
}
