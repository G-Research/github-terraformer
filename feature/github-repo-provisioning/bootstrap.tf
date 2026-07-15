import {
  for_each = var.bootstrap ? local.teams_by_name : {}

  to = github_team.team[each.key]
  id = try(each.value.slug, each.key)
}

import {
  for_each = var.bootstrap ? local.members_by_username : {}

  to = github_membership.member[each.key]
  id = "${var.owner}:${each.key}"
}

import {
  for_each = var.bootstrap ? local.member_team_pairs : {}

  to = github_team_membership.membership[each.key]
  id = "${try(local.teams_by_name[each.value.team].slug, each.value.team)}:${each.value.username}"
}
