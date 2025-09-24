provider "github" {
  owner = var.owner
  app_auth {
    id = var.app_id
    installation_id = var.app_installation_id
    pem_file = var.app_private_key
  }
}

locals {
  generated_repos = merge(
    {
      for file_path in fileset(path.module, "gcss_config/importer_tmp_dir/*.yaml") :
      split(".yaml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "gcss_config/importer_tmp_dir/*.yml") :
      split(".yml", basename(file_path))[0] => yamldecode(file(file_path))
    }
  )
  new_repos = merge(
    {
      for file_path in fileset(path.module, "gcss_config/*.yaml") :
      split(".yaml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "gcss_config/*.yml") :
      split(".yml", basename(file_path))[0] => yamldecode(file(file_path))
    }
  )

  all_repos = merge(local.generated_repos, local.new_repos)
}

import {
  for_each = local.generated_repos
  to = module.repository[each.key].github_repository.repository
  id = each.key
}

import {
  for_each = local.generated_repos
  to = module.repository[each.key].github_branch_default.default[0]
  id = each.key
}

locals {
  flattened_generated_branch_protections_v4 = flatten([
    for repo, config in local.generated_repos : [
      for branch_protection in try(config.branch_protections_v4, []) : {
        repository = repo
        branch_protection = branch_protection
      }
    ]
  ])
}

import {
  for_each = local.flattened_generated_branch_protections_v4

  to = module.repository[each.value.repository].github_branch_protection.branch_protection[each.value.branch_protection.pattern]
  id = format("%s:%s", each.value.repository, each.value.branch_protection.pattern)
}

locals {
  flat_restrictions = distinct(flatten(
    concat(
      try([for p in local.flattened_generated_branch_protections_v4 : try(p.branch_protection.push_restrictions, [])], []),
      try([for p in local.flattened_generated_branch_protections_v4 : try(p.branch_protection.required_pull_request_reviews.dismissal_restrictions, [])], []),
      try([for p in local.flattened_generated_branch_protections_v4 : try(p.branch_protection.required_pull_request_reviews.pull_request_bypassers, [])], []),
      try([for p in local.flattened_generated_branch_protections_v4 : try(p.branch_protection.force_push_bypassers, [])], [])
    )
  ))
  app_actors = [for actor in local.flat_restrictions : actor if startswith(actor, "app/")]
}

data "github_app" "app" {
  for_each = toset(local.app_actors)
  slug = split("/", each.value)[1]
}

locals {
  all_generated_collaborators = { for repo, config in local.generated_repos : repo => concat(
    try([for i in config.pull_collaborators     : { username: i,  permission = "pull"     }], []),
    try([for i in config.push_collaborators     : { username: i,  permission = "push"     }], []),
    try([for i in config.admin_collaborators    : { username: i,  permission = "admin"    }], []),
    try([for i in config.maintain_collaborators : { username: i,  permission = "maintain" }], []),
    try([for i in config.triage_collaborators   : { username: i,  permission = "triage"   }], [])
  )}

  all_generated_teams = { for repo, config in local.generated_repos : repo => concat(
    try([for i in config.pull_teams     : { name: i,  permission = "pull"     }], []),
    try([for i in config.push_teams     : { name: i,  permission = "push"     }], []),
    try([for i in config.admin_teams    : { name: i,  permission = "admin"    }], []),
    try([for i in config.maintain_teams : { name: i,  permission = "maintain" }], []),
    try([for i in config.triage_teams   : { name: i,  permission = "triage"   }], [])
  )}

  all_new_collaborators = { for repo, config in local.new_repos : repo => concat(
    try([for i in config.pull_collaborators     : { username: i,  permission = "pull"     }], []),
    try([for i in config.push_collaborators     : { username: i,  permission = "push"     }], []),
    try([for i in config.admin_collaborators    : { username: i,  permission = "admin"    }], []),
    try([for i in config.maintain_collaborators : { username: i,  permission = "maintain" }], []),
    try([for i in config.triage_collaborators   : { username: i,  permission = "triage"   }], [])
  )}

  all_new_teams = { for repo, config in local.new_repos : repo => concat(
    try([for i in config.pull_teams     : { name: i,  permission = "pull"     }], []),
    try([for i in config.push_teams     : { name: i,  permission = "push"     }], []),
    try([for i in config.admin_teams    : { name: i,  permission = "admin"    }], []),
    try([for i in config.maintain_teams : { name: i,  permission = "maintain" }], []),
    try([for i in config.triage_teams   : { name: i,  permission = "triage"   }], [])
  )}

  all_collaborators = merge(local.all_generated_collaborators, local.all_new_collaborators)
  all_teams = merge(local.all_generated_teams, local.all_new_teams)
}

import {
  for_each = toset(flatten([for repo, collaborators in local.all_generated_collaborators : [
    for collaborator in collaborators : {
      repo      = repo
      username  = collaborator.username
      permission = collaborator.permission
    }
  ]]))

  to = module.repository[each.value.repo].github_repository_collaborator.collaborator[each.value.username]
  id = "${each.value.repo}:${each.value.username}"
}

data "github_team" "team" {
  for_each = toset(distinct(flatten([
    for repo, teams in local.all_teams : [
      for team in teams : team.name
    ]
  ])))
  slug = each.value
}

import {
  for_each = toset(flatten([for repo, teams in local.all_generated_teams : [
    for team in teams : {
      repo        = repo
      name        = team.name
      team_id     = data.github_team.team[team.name].id
    }
  ]]))

  to = module.repository[each.value.repo].github_team_repository.team_repository_by_slug[each.value.team_id]
  id = "${each.value.team_id}:${each.value.repo}"
}


module "repository" {
  source                  = "./modules/terraform-github-repository"
  for_each                = local.all_repos

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Main resource configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  name                    = each.key
  allow_merge_commit      = try(each.value.allow_merge_commit,      true)
  allow_rebase_merge      = try(each.value.allow_rebase_merge,      false)
  allow_squash_merge      = try(each.value.allow_squash_merge,      false)
  allow_auto_merge        = try(each.value.allow_auto_merge,        false)
  allow_update_branch     = try(each.value.allow_update_branch,     null)
  description             = try(each.value.description,             "")
  delete_branch_on_merge  = try(each.value.delete_branch_on_merge,  true)
  homepage_url            = try(each.value.homepage_url,            "")
  visibility              = try(each.value.visibility,              "private")
  has_issues              = try(each.value.has_issues,              false)
  has_projects            = try(each.value.has_projects,            false)
  has_wiki                = try(each.value.has_wiki,                false)
  has_downloads           = try(each.value.has_downloads,           false)
  has_discussions         = try(each.value.has_discussions,         null)
  is_template             = try(each.value.is_template,             false)
  default_branch          = try(each.value.default_branch,          "")
  archived                = try(each.value.archived,                false)
  topics                  = try(each.value.topics,                  [])
  archive_on_destroy      = try(each.value.archive_on_destroy,      null)
  pages                   = try(contains(keys(each.value), "pages") && try(each.value.pages != null, false) ? {
                              branch      = try(each.value.pages.branch, "gh-pages")
                              path        = try(each.value.pages.path,   "/")
                              cname       = try(each.value.pages.cname,  null)
                              build_type  = try(each.value.pages.build_type,  null)
                            } : null)
  vulnerability_alerts    = try(each.value.vulnerability_alerts_enabled,  null)

  squash_merge_commit_title   = try(each.value.squash_merge_commit_title,   null)
  squash_merge_commit_message = try(each.value.squash_merge_commit_message, null)
  merge_commit_title          = try(each.value.merge_commit_title,          null)
  merge_commit_message        = try(each.value.merge_commit_message,        null)
  web_commit_signoff_required = try(each.value.web_commit_signoff_required, null)

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Extended Resource Configuration
  # Repository Creation Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  auto_init           = try(each.value.auto_init,                           true)
  gitignore_template  = try(each.value.gitignore_template,                  "")
  license_template    = try(each.value.license_template,                    "")
  template            = try(contains(keys(each.value), "template") && try(each.value.template != null, false) ? {
                          owner       = try(each.value.template.owner,      "")
                          repository  = try(each.value.template.repository, "")
                        } : null)

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Teams Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  pull_teams      = try([for i in each.value.pull_teams     : data.github_team.team[i].id],  [])
  push_teams      = try([for i in each.value.push_teams     : data.github_team.team[i].id],  [])
  admin_teams     = try([for i in each.value.admin_teams    : data.github_team.team[i].id],  [])
  maintain_teams  = try([for i in each.value.maintain_teams : data.github_team.team[i].id],  [])
  triage_teams    = try([for i in each.value.triage_teams   : data.github_team.team[i].id],  [])

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Collaborator Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  pull_collaborators      = try(each.value.pull_collaborators,      [])
  push_collaborators      = try(each.value.push_collaborators,      [])
  admin_collaborators     = try(each.value.admin_collaborators,     [])
  maintain_collaborators  = try(each.value.maintain_collaborators,  [])
  triage_collaborators    = try(each.value.triage_collaborators,    [])

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Branches Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Deploy Keys Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Branch Protections v3 Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Branch Protections v4 Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  branch_protections_v4 = try([
    for branch_protection in try(each.value.branch_protections_v4, []) : {
      pattern                         = branch_protection.pattern
      allows_deletions                = try(branch_protection.allows_deletions, false)
      allows_force_pushes             = try(branch_protection.allows_force_pushes, false)
      force_push_bypassers            = try([for bypasser in branch_protection.force_push_bypassers : (!startswith(bypasser, "app/") ? bypasser : data.github_app.app[bypasser].node_id)], [])
      enforce_admins                  = try(branch_protection.enforce_admins, true)
      lock_branch                     = try(branch_protection.lock_branch, null)

      restricts_pushes                = try(branch_protection.restricts_pushes, false)
      blocks_creations                = try(branch_protection.blocks_creations, false)
      push_restrictions               = try([for bypasser in branch_protection.push_restrictions : (!startswith(bypasser, "app/") ? bypasser : data.github_app.app[bypasser].node_id)], [])

      require_conversation_resolution = try(branch_protection.require_conversation_resolution, false)
      require_signed_commits          = try(branch_protection.require_signed_commits, false)
      required_linear_history         = try(branch_protection.required_linear_history, false)

      required_pull_request_reviews = try(branch_protection.required_pull_request_reviews, null) == null ? null : {
        required_approving_review_count = try(branch_protection.required_pull_request_reviews.required_approving_review_count, 0)
        dismiss_stale_reviews           = try(branch_protection.required_pull_request_reviews.dismiss_stale_reviews, true)
        require_code_owner_reviews      = try(branch_protection.required_pull_request_reviews.require_code_owner_reviews, true)
        restrict_dismissals             = try(branch_protection.required_pull_request_reviews.restrict_dismissals, false)
        pull_request_bypassers          = try([for bypasser in branch_protection.required_pull_request_reviews.pull_request_bypassers : (!startswith(bypasser, "app/") ? bypasser : data.github_app.app[bypasser].node_id)], [])
        dismissal_restrictions          = try([for bypasser in branch_protection.required_pull_request_reviews.dismissal_restrictions : (!startswith(bypasser, "app/") ? bypasser : data.github_app.app[bypasser].node_id)], [])
        require_last_push_approval      = try(branch_protection.required_pull_request_reviews.require_last_push_approval, null)
      }

      required_status_checks = try(branch_protection.required_status_checks, null) == null ? null : {
        strict   = try(branch_protection.required_status_checks.strict, false)
        contexts = try(branch_protection.required_status_checks.contexts, [])
      }
    }
  ], [])

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Issue Labels Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  issue_labels_create = false

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Projects Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Webhooks Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Secrets Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # Autolink References Configuration
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  # App Installations
  # ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

#  app_installations = try(each.value.app_installations, [])
}

locals {
  new_rulesets_flattened = flatten([
    for repo, config in local.new_repos : [
      for ruleset in try(config.rulesets, []) : {
        repository  = repo
        ruleset     = ruleset
      }
    ]
  ])

  new_rulesets_map = {
    for idx, item in local.new_rulesets_flattened :
    sha256("${item.repository}/${item.ruleset.name}") => item
  }

  generated_rulesets_flattened = flatten([
    for repo, config in local.generated_repos : [
      for ruleset in try(config.rulesets, []) : {
        repository  = repo
        ruleset     = ruleset
      }
    ]
  ])

  generated_rulesets_map = {
    for idx, item in local.generated_rulesets_flattened :
    sha256("${item.repository}/${item.ruleset.name}") => item
  }

  all_rulesets_map = merge(local.new_rulesets_map, local.generated_rulesets_map)
}

import {
  for_each = local.generated_rulesets_map
  to = github_repository_ruleset.ruleset[each.key]
  id = format("%s:%s", each.value.repository, each.value.ruleset.id)
}

locals {
  ruleset_actors = {
    "repository-admin-role" = {
      actor_type = "RepositoryRole"
      actor_id   = 5
    }
    "organization-admin-role" = {
      actor_type = "OrganizationAdmin"
      actor_id   = 1
    }
    "maintain-role" = {
      actor_type = "RepositoryRole"
      actor_id   = 2
    }
    "write-role" = {
      actor_type = "RepositoryRole"
      actor_id   = 4
    }
  }

  apps_map = {
    for app in yamldecode(file("./app-list.yaml")).apps :
    "app/${app.app_owner}/${app.app_slug}" => {
      app_id = app.app_id
    }
  }

  team_bypass_actors = distinct(flatten([
    for v in values(local.all_rulesets_map) : [
      for actor in try(v.ruleset.bypass_actors, []) : replace(actor.name, "team/", "")
      if startswith(actor.name, "team/")
    ]
  ]))
}

data "github_team" "ruleset_team" {
  for_each = toset(local.team_bypass_actors)
  slug = each.value
}

resource "github_repository_ruleset" "ruleset" {
  depends_on = [module.repository]

  for_each  = local.all_rulesets_map
  name      = each.value.ruleset.name
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
            context       = required_check.value.context
            integration_id = required_check.value.integration_id
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
        ? [each.value.ruleset.rules.required_code_scanning]  # Only one block for `required_code_scanning`
        : []
      )

      content {
        dynamic "required_code_scanning_tool" {
          for_each = try(each.value.ruleset.rules.required_code_scanning.required_code_scanning_tool, [])

          content {
            tool                    = required_code_scanning_tool.value.tool
            alerts_threshold        = required_code_scanning_tool.value.alerts_threshold
            security_alerts_threshold = required_code_scanning_tool.value.security_alerts_threshold
          }
        }
      }
    }
  }

  dynamic "bypass_actors" {
    for_each = try(each.value.ruleset.bypass_actors, [])

    content {
      actor_id    = startswith(bypass_actors.value.name, "team/") ? data.github_team.ruleset_team[replace(bypass_actors.value.name, "team/", "")].id : (
        startswith(bypass_actors.value.name, "app/") ? local.apps_map[bypass_actors.value.name].app_id : local.ruleset_actors[bypass_actors.value.name].actor_id
      )
      actor_type  = startswith(bypass_actors.value.name, "team/") ? "Team" : (
        startswith(bypass_actors.value.name, "app/") ? "Integration" : local.ruleset_actors[bypass_actors.value.name].actor_type
      )
      bypass_mode = try(bypass_actors.value.bypass_mode,  "always")
    }
  }
}
