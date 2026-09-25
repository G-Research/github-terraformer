locals {
  teams_config_files        = fileset(path.module, "gcss_config/organisation/teams.yaml")
  staged_teams_config_files = fileset(path.module, "gcss_config/importer_tmp_dir/organisation/teams.yaml")

  teams_raw        = length(local.teams_config_files) > 0 ? yamldecode(file("${path.module}/gcss_config/organisation/teams.yaml")) : { teams = [] }
  staged_teams_raw = length(local.staged_teams_config_files) > 0 ? yamldecode(file("${path.module}/gcss_config/importer_tmp_dir/organisation/teams.yaml")) : { teams = [] }

  staged_teams_by_name = { for t in try(local.staged_teams_raw.teams, []) : t.name => t }

  teams_by_name = merge(
    local.staged_teams_by_name,
    { for t in try(local.teams_raw.teams, []) : t.name => t },
  )

  # Split teams by whether they nest under a parent. Two resources let a child reference its
  # (root) parent's id for correct create ordering, which a single for_each resource can't do
  # (a self-reference across instances cycles). Supports one level of nesting.
  root_teams_by_name  = { for k, t in local.teams_by_name : k => t if try(t.parent, null) == null }
  child_teams_by_name = { for k, t in local.teams_by_name : k => t if try(t.parent, null) != null }
  # Bucket staged teams by the MERGED result (final config wins), not the staged file's own
  # parent — otherwise, when a team's parent differs between staged and final (e.g. re-import
  # after a parent change on GitHub), the import target address wouldn't match the resource.
  staged_root_teams  = { for k, t in local.staged_teams_by_name : k => t if contains(keys(local.root_teams_by_name), k) }
  staged_child_teams = { for k, t in local.staged_teams_by_name : k => t if contains(keys(local.child_teams_by_name), k) }

  # Team name -> id across both resources, for github_team_membership and any other lookups.
  team_ids = merge(
    { for k, t in github_team.team : k => t.id },
    { for k, t in github_team.child_team : k => t.id },
  )
}

import {
  for_each = local.staged_root_teams

  to = github_team.team[each.key]
  id = try(each.value.slug, each.key)
}

import {
  for_each = local.staged_child_teams

  to = github_team.child_team[each.key]
  id = try(each.value.slug, each.key)
}

resource "github_team" "team" {
  for_each = local.root_teams_by_name

  name                 = each.value.name
  description          = try(each.value.description, null)
  privacy              = try(each.value.visibility, "visible") == "secret" ? "secret" : "closed"
  notification_setting = coalesce(try(each.value.notifications, true), true) ? "notifications_enabled" : "notifications_disabled"
}

resource "github_team" "child_team" {
  for_each = local.child_teams_by_name

  name                 = each.value.name
  description          = try(each.value.description, null)
  privacy              = try(each.value.visibility, "visible") == "secret" ? "secret" : "closed"
  notification_setting = coalesce(try(each.value.notifications, true), true) ? "notifications_enabled" : "notifications_disabled"
  # Parent referenced by name; it must be a root team (github_team.team) — one level of nesting.
  # The reference gives Terraform the parent-before-child ordering without a self-cycle.
  parent_team_id = github_team.team[each.value.parent].id
}
