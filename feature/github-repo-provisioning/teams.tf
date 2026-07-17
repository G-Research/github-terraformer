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
}

import {
  for_each = local.staged_teams_by_name

  to = github_team.team[each.key]
  id = try(each.value.slug, each.key)
}

resource "github_team" "team" {
  for_each = local.teams_by_name

  name                 = each.value.name
  description          = try(each.value.description, null)
  privacy              = try(each.value.visibility, "visible") == "secret" ? "secret" : "closed"
  notification_setting = coalesce(try(each.value.notifications, true), true) ? "notifications_enabled" : "notifications_disabled"
}
