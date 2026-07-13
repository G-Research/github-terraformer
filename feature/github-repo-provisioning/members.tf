locals {
  members_config_files = fileset(path.module, "gcss_config/organisation/members.yaml")

  members_content = length(local.members_config_files) > 0 ? file("${path.module}/gcss_config/organisation/members.yaml") : "members: []"

  members_raw = yamldecode(trimspace(local.members_content) == "" ? "members: []" : local.members_content)

  members_list = try([for m in local.members_raw.members : m if m != null], [])

  members_by_username = { for m in local.members_list : m.username => m }

  member_team_pairs = {
    for pair in flatten([
      for username, m in local.members_by_username : try([
        for team in m.teams : {
          username = username
          team     = team
        }
      ], [])
    ]) : "${pair.username}/${pair.team}" => pair
  }
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
  role     = "member"
}
