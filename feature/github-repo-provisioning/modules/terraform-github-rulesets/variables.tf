variable "rulesets" {
  description = "(Required) Map of rulesets to manage. Keys are sha256 hashes of 'repository/ruleset_name'."
  type        = any
  default     = {}
}

variable "apps_map" {
  description = "(Optional) Map of app path keys to app_id objects, used for app bypass actors and status check integration IDs."
  type        = map(object({ app_id = number }))
  default     = {}
}

variable "ruleset_actors" {
  description = "(Optional) Map of named role keys to actor_type/actor_id pairs used in bypass_actors."
  type = map(object({
    actor_type = string
    actor_id   = number
  }))
  default = {
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
}

variable "builtin_github_sources" {
  description = "(Optional) Map of built-in GitHub source names to their integration IDs."
  type        = map(number)
  default = {
    "Any source"     = 0
    "GitHub Actions" = 15368
  }
}
