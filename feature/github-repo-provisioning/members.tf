locals {
  members_config_files        = fileset(path.module, "gcss_config/organisation/members.yaml")
  staged_members_config_files = fileset(path.module, "gcss_config/importer_tmp_dir/organisation/members.yaml")

  members_content        = length(local.members_config_files) > 0 ? file("${path.module}/gcss_config/organisation/members.yaml") : "members: []"
  staged_members_content = length(local.staged_members_config_files) > 0 ? file("${path.module}/gcss_config/importer_tmp_dir/organisation/members.yaml") : "members: []"

  members_raw        = yamldecode(trimspace(local.members_content) == "" ? "members: []" : local.members_content)
  staged_members_raw = yamldecode(trimspace(local.staged_members_content) == "" ? "members: []" : local.staged_members_content)

  members_list        = try([for m in local.members_raw.members : m if m != null], [])
  staged_members_list = try([for m in local.staged_members_raw.members : m if m != null], [])

  staged_members_by_username = { for m in local.staged_members_list : m.username => m }

  members_by_username = merge(
    local.staged_members_by_username,
    { for m in local.members_list : m.username => m },
  )

  member_team_pairs = {
    for pair in flatten([
      for username, m in local.members_by_username : try([
        for team in m.teams : {
          username = username
          team     = team.name
          role     = try(team.role, "member")
        }
      ], [])
    ]) : "${pair.username}/${pair.team}" => pair
  }

  staged_member_team_pairs = {
    for key, pair in local.member_team_pairs : key => pair
    if contains(keys(local.staged_members_by_username), pair.username)
  }
}

import {
  for_each = local.staged_members_by_username

  to = github_membership.member[each.key]
  id = "${var.owner}:${each.key}"
}

import {
  for_each = local.staged_member_team_pairs

  to = github_team_membership.membership[each.key]
  id = "${try(local.teams_by_name[each.value.team].slug, each.value.team)}:${each.value.username}"
}

resource "github_membership" "member" {
  for_each = local.members_by_username

  username = each.value.username
  role     = try(each.value.role, "member") == "owner" ? "admin" : "member"
}

resource "github_team_membership" "membership" {
  for_each = local.member_team_pairs

  team_id  = github_team.team[each.value.team].id
  username = each.value.username
  role     = each.value.role
}
