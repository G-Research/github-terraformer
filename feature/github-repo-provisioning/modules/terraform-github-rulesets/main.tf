locals {
  team_bypass_actors = distinct(flatten([
    for v in values(var.rulesets) : [
      for actor in try(v.ruleset.bypass_actors, []) : replace(actor.name, "team/", "")
      if startswith(actor.name, "team/")
    ]
  ]))
}

data "github_team" "ruleset_team" {
  for_each = toset(local.team_bypass_actors)
  slug     = each.value
}

resource "github_repository_ruleset" "ruleset" {
  for_each    = var.rulesets
  name        = each.value.ruleset.name
  enforcement = each.value.ruleset.enforcement
  target      = each.value.ruleset.target
  repository  = each.value.repository


  dynamic "conditions" {
    for_each = try(each.value.ruleset.conditions, null) != null ? [each.value.ruleset.conditions] : []

    content {
      ref_name {
        include = try(each.value.ruleset.conditions.ref_name.include, [])
        exclude = try(each.value.ruleset.conditions.ref_name.exclude, [])
      }
    }
  }

  rules {
    creation                      = try(each.value.ruleset.rules.creation, null)
    update                        = try(each.value.ruleset.rules.update, null)
    deletion                      = try(each.value.ruleset.rules.deletion, null)
    required_linear_history       = try(each.value.ruleset.rules.required_linear_history, null)
    required_signatures           = try(each.value.ruleset.rules.required_signatures, null)
    non_fast_forward              = try(each.value.ruleset.rules.non_fast_forward, null)
    update_allows_fetch_and_merge = try(each.value.ruleset.rules.update_allows_fetch_and_merge, null)

    dynamic "branch_name_pattern" {
      for_each = try(each.value.ruleset.rules.branch_name_pattern, null) != null ? [each.value.ruleset.rules.branch_name_pattern] : []

      content {
        name     = try(each.value.ruleset.rules.branch_name_pattern.name, null)
        operator = each.value.ruleset.rules.branch_name_pattern.operator
        pattern  = each.value.ruleset.rules.branch_name_pattern.pattern
        negate   = try(each.value.ruleset.rules.branch_name_pattern.negate, null)
      }
    }

    dynamic "tag_name_pattern" {
      for_each = try(each.value.ruleset.rules.tag_name_pattern, null) != null ? [each.value.ruleset.rules.tag_name_pattern] : []

      content {
        name     = try(each.value.ruleset.rules.tag_name_pattern.name, null)
        operator = each.value.ruleset.rules.tag_name_pattern.operator
        pattern  = each.value.ruleset.rules.tag_name_pattern.pattern
        negate   = try(each.value.ruleset.rules.tag_name_pattern.negate, null)
      }
    }

    dynamic "commit_author_email_pattern" {
      for_each = try(each.value.ruleset.rules.commit_author_email_pattern, null) != null ? [each.value.ruleset.rules.commit_author_email_pattern] : []

      content {
        name     = try(each.value.ruleset.rules.commit_author_email_pattern.name, null)
        operator = each.value.ruleset.rules.commit_author_email_pattern.operator
        pattern  = each.value.ruleset.rules.commit_author_email_pattern.pattern
        negate   = try(each.value.ruleset.rules.commit_author_email_pattern.negate, null)
      }
    }

    dynamic "commit_message_pattern" {
      for_each = try(each.value.ruleset.rules.committer_email_pattern, null) != null ? [each.value.ruleset.rules.committer_email_pattern] : []

      content {
        name     = try(each.value.ruleset.rules.committer_email_pattern.name, null)
        operator = each.value.ruleset.rules.committer_email_pattern.operator
        pattern  = each.value.ruleset.rules.committer_email_pattern.pattern
        negate   = try(each.value.ruleset.rules.committer_email_pattern.negate, null)
      }
    }

    dynamic "pull_request" {
      for_each = try(each.value.ruleset.rules.pull_request, null) != null ? [each.value.ruleset.rules.pull_request] : []

      content {
        dismiss_stale_reviews_on_push     = try(each.value.ruleset.rules.pull_request.dismiss_stale_reviews_on_push, null)
        require_code_owner_review         = try(each.value.ruleset.rules.pull_request.require_code_owner_review, null)
        require_last_push_approval        = try(each.value.ruleset.rules.pull_request.require_last_push_approval, null)
        required_approving_review_count   = try(each.value.ruleset.rules.pull_request.required_approving_review_count, null)
        required_review_thread_resolution = try(each.value.ruleset.rules.pull_request.required_review_thread_resolution, null)
      }
    }

    dynamic "required_status_checks" {
      for_each = (
        contains(keys(each.value.ruleset.rules), "required_status_checks") &&
        try(each.value.ruleset.rules.required_status_checks != null, false) &&
        length(try(each.value.ruleset.rules.required_status_checks.required_check, [])) > 0
      ) ? [each.value.ruleset.rules.required_status_checks] : []

      content {
        strict_required_status_checks_policy = try(required_status_checks.value.strict_required_status_checks_policy, null)

        dynamic "required_check" {
          for_each = try(required_status_checks.value.required_check, [])
          content {
            context        = required_check.value.context
            integration_id = (startswith(required_check.value.source, "app/") ? var.apps_map[required_check.value.source].app_id : var.builtin_github_sources[required_check.value.source])
          }
        }
      }
    }

    dynamic "required_deployments" {
      for_each = try(
        contains(keys(each.value.ruleset.rules), "required_deployments") &&
        try(each.value.ruleset.rules.required_deployments != null, false) &&
        try(length(keys(each.value.ruleset.rules.required_deployments)) > 0, false)
        ? [each.value.ruleset.rules.required_deployments]
        : []
      )

      content {
        required_deployment_environments = try(required_deployments.value.required_deployment_environments, ["staging", "production"])
      }
    }

    dynamic "required_code_scanning" {
      for_each = try(
        contains(keys(each.value.ruleset.rules), "required_code_scanning") &&
        try(each.value.ruleset.rules.required_code_scanning != null, false) &&
        length(try(each.value.ruleset.rules.required_code_scanning.required_code_scanning_tool, [])) > 0
        ? [each.value.ruleset.rules.required_code_scanning]
        : []
      )

      content {
        dynamic "required_code_scanning_tool" {
          for_each = try(each.value.ruleset.rules.required_code_scanning.required_code_scanning_tool, [])

          content {
            tool                      = required_code_scanning_tool.value.tool
            alerts_threshold          = required_code_scanning_tool.value.alerts_threshold
            security_alerts_threshold = required_code_scanning_tool.value.security_alerts_threshold
          }
        }
      }
    }
  }

  dynamic "bypass_actors" {
    for_each = try(each.value.ruleset.bypass_actors, [])

    content {
      actor_id = startswith(bypass_actors.value.name, "team/") ? data.github_team.ruleset_team[replace(bypass_actors.value.name, "team/", "")].id : (
        startswith(bypass_actors.value.name, "app/") ? var.apps_map[bypass_actors.value.name].app_id : var.ruleset_actors[bypass_actors.value.name].actor_id
      )
      actor_type = startswith(bypass_actors.value.name, "team/") ? "Team" : (
        startswith(bypass_actors.value.name, "app/") ? "Integration" : var.ruleset_actors[bypass_actors.value.name].actor_type
      )
      bypass_mode = try(bypass_actors.value.bypass_mode, "always")
    }
  }
}
