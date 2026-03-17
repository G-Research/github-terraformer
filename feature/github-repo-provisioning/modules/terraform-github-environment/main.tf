# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
# MANAGE A GITHUB DEPLOYMENT ENVIRONMENT
# Creates a github_repository_environment and, when custom branch/tag policies are used,
# one github_repository_environment_deployment_policy per pattern.
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

# Look up reviewer users so we can pass their numeric IDs to the reviewers block.
data "github_user" "reviewer_user" {
  for_each = toset(var.reviewer_users)
  username = each.value
}

# Look up reviewer teams so we can pass their numeric IDs to the reviewers block.
data "github_team" "reviewer_team" {
  for_each = toset(var.reviewer_teams)
  slug     = each.value
}

resource "github_repository_environment" "environment" {
  repository          = var.repository
  environment         = var.environment
  wait_timer          = var.wait_timer
  can_admins_bypass   = var.can_admins_bypass
  prevent_self_review = var.prevent_self_review

  dynamic "reviewers" {
    for_each = (length(var.reviewer_users) + length(var.reviewer_teams)) > 0 ? [1] : []
    content {
      users = [for u in var.reviewer_users : data.github_user.reviewer_user[u].id]
      teams = [for t in var.reviewer_teams : data.github_team.reviewer_team[t].id]
    }
  }

  dynamic "deployment_branch_policy" {
    for_each = var.deployment_policy != null && var.deployment_policy.policy_type != "all" ? [var.deployment_policy] : []
    content {
      protected_branches     = deployment_branch_policy.value.policy_type == "protected_branches"
      custom_branch_policies = deployment_branch_policy.value.policy_type == "selected_branches_and_tags"
    }
  }
}

# One resource per branch/tag pattern when using custom branch policies.
resource "github_repository_environment_deployment_policy" "branch_pattern" {
  for_each = toset(
    try(
      var.deployment_policy.policy_type == "selected_branches_and_tags" ? var.deployment_policy.branch_patterns : [],
      []
    )
  )

  repository     = var.repository
  environment    = github_repository_environment.environment.environment
  branch_pattern = each.value
}
