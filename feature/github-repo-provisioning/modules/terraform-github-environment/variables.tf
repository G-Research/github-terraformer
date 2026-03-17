# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
# INPUT VARIABLES
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

variable "repository" {
  description = "The name of the repository the environment belongs to."
  type        = string
}

variable "environment" {
  description = "The name of the deployment environment."
  type        = string
}

variable "wait_timer" {
  description = "How many minutes to wait before deploying to this environment. Null means no wait."
  type        = number
  default     = null
}

variable "can_admins_bypass" {
  description = "Whether repository admins can bypass environment protection rules. Null uses the GitHub default (true)."
  type        = bool
  default     = null
}

variable "prevent_self_review" {
  description = "Whether the user who triggered the workflow run can approve their own deployment. Null uses the GitHub default (false)."
  type        = bool
  default     = null
}

variable "reviewer_users" {
  description = "List of GitHub usernames that must approve deployments to this environment (up to 6 total across users and teams)."
  type        = list(string)
  default     = []
}

variable "reviewer_teams" {
  description = "List of GitHub team slugs that must approve deployments to this environment (up to 6 total across users and teams)."
  type        = list(string)
  default     = []
}

variable "deployment_policy" {
  description = <<-EOT
    Controls which branches and tags can deploy to this environment.
    - null / omitted  → all branches (no restriction)
    - policy_type = "protected_branches"         → only branches with branch protection rules
    - policy_type = "selected_branches_and_tags" → only branches/tags matching branch_patterns
  EOT
  type = object({
    policy_type     = string
    branch_patterns = optional(list(string), [])
  })
  default = null

  validation {
    condition = var.deployment_policy == null || contains(
      ["all", "protected_branches", "selected_branches_and_tags"],
      var.deployment_policy.policy_type
    )
    error_message = "deployment_policy.policy_type must be one of: all, protected_branches, selected_branches_and_tags."
  }
}
