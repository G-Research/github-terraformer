package github

const (
	// Rule types
	RuleTypeRequiredLinearHistory = "required_linear_history"
	RuleTypePullRequest           = "pull_request"
	RuleTypeRequiredStatusChecks  = "required_status_checks"
	RuleTypeDeletion              = "deletion"
	RuleTypeCreation              = "creation"
	RuleTypeNonFastForward        = "non_fast_forward"
	RuleRequiredSignatures        = "required_signatures"
	RuleUpdate                    = "update"
	RuleRequiredDeployments       = "required_deployments"
	RuleCommitMessagePattern      = "commit_message_pattern"
	RuleCommitAuthorEmailPattern  = "commit_author_email_pattern"
	RuleCommitterEmailPattern     = "committer_email_pattern"
	RuleBranchNamePattern         = "branch_name_pattern"
	RuleTagNamePattern            = "tag_name_pattern"
	RuleCodeScanning              = "code_scanning"

	// Visibility
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"

	// Permission levels
	PermissionRead     = "read"
	PermissionWrite    = "write"
	PermissionPull     = "pull"
	PermissionPush     = "push"
	PermissionTriage   = "triage"
	PermissionMaintain = "maintain"
	PermissionAdmin    = "admin"

	DefaultPageSize = 100

	BypassActorType_RepositoryRole            = "RepositoryRole"
	BypassActorType_OrganizationAdmin         = "OrganizationAdmin"
	BypassActorType_Team                      = "Team"
	BypassActorType_Integration               = "Integration"
	BypassActorType_DeployKey                 = "DeployKey"
	BypassActorId_OrganizationAdminRole       = 1
	BypassActorId_RepositoryAdminRole         = 5
	BypassActorId_MaintainRole                = 2
	BypassActorId_WriteRole                   = 4
	BypassActorRoleName_RepositoryAdminRole   = "repository-admin-role"
	BypassActorRoleName_OrganizationAdminRole = "organization-admin-role"
	BypassActorRoleName_MaintainRole          = "maintain-role"
	BypassActorRoleName_WriteRole             = "write-role"
)
