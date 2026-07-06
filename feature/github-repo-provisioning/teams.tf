locals {
  teams_config_files = fileset(path.module, "gcss_config/organisation/teams.yaml")

  teams_raw = length(local.teams_config_files) > 0 ? yamldecode(file("${path.module}/gcss_config/organisation/teams.yaml")) : { teams = [] }

  teams_by_name = { for t in try(local.teams_raw.teams, []) : t.name => t }
}

resource "github_team" "team" {
  for_each = local.teams_by_name

  name                 = each.value.name
  description          = try(each.value.description, null)
  privacy              = try(each.value.visibility, "visible") == "secret" ? "secret" : "closed"
  notification_setting = coalesce(try(each.value.notifications, true), true) ? "notifications_enabled" : "notifications_disabled"
  parent_team_id       = try(each.value.parent, null) != null ? github_team.team[each.value.parent].id : null
}
