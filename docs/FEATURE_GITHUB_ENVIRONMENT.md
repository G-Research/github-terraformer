# GitHub Environments Configuration Guide

This guide explains how to configure GitHub repository environments using the YAML → Terraform workflow.

## Quick Start

### Enable Environment Import
```yaml
# gcss-config-repo/config/import-config.yaml
feature_github_environment: true  # Required to import environments
```

### Environment Configuration

```yaml
# repos/my-app.yaml
environments:
  # Option 1: Protected branches only
  - environment: production
    wait_timer: 300               # 5 minutes wait before deployment
    can_admins_bypass: false      # Admins cannot bypass
    prevent_self_review: true     # Cannot approve own deployments
    reviewers:
      users: ["octocat"]
      teams: ["platform-team"]
    deployment_policy:
      policy_type: protected_branches

  # Option 2: Custom branch/tag patterns
  - environment: staging
    deployment_policy:
      policy_type: selected_branches_and_tags
      branch_patterns:
        - "release/*"
        - "main"
      tag_patterns:
        - "v*"

  # Option 3: Any branch can deploy (no restrictions)
  - environment: development
    # No deployment_policy = any branch can deploy
```

## ⚠️ Critical Rule: Deployment Policy Types

**You MUST choose ONE of these options:**

| Option | Configuration | Use Case |
|--------|--------------|----------|
| **Protected Branches** | `policy_type: protected_branches` | Production - only protected branches |
| **Custom Patterns** | `policy_type: selected_branches_and_tags` + patterns | Staging - specific branches/tags |
| **Any Branch** | Omit `deployment_policy` entirely | Development - no restrictions |

**The `policy_type` field determines which patterns are used.**

## Field Reference

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `environment` | string | **Required** - Environment name | - |
| `wait_timer` | int | Wait time in seconds (max 43200) | 0 |
| `can_admins_bypass` | bool | Admins can bypass protections | true |
| `prevent_self_review` | bool | Prevent self-approval | false |
| `reviewers.users` | string[] | GitHub usernames (max 6 total with teams) | [] |
| `reviewers.teams` | string[] | Team slugs (max 6 total with users) | [] |
| `deployment_policy.*` | object | Deployment restrictions | - |
| ↳ `policy_type` | string | `protected_branches` or `selected_branches_and_tags` | - |
| ↳ `branch_patterns` | string[] | Branch patterns (only for `selected_branches_and_tags`) | [] |
| ↳ `tag_patterns` | string[] | Tag patterns (only for `selected_branches_and_tags`) | [] |

## Pattern Matching

Patterns support wildcards:
- `main` - Exact match
- `release/*` - Matches `release/1.0`, `release/2.0`
- `v*` - Matches `v1.0.0`, `v2.0.0`
- `*-final` - Matches `1.0-final`, `2.0-final`

## Generated Terraform Resources

The YAML configuration generates:

1. **Environment Resource**
```hcl
resource "github_repository_environment" "environment" {
  environment = "production"
  repository  = "my-app"
  # ... other settings

  deployment_branch_policy {
    protected_branches     = true/false
    custom_branch_policies = true/false
  }
}
```

2. **Deployment Policies** (for custom patterns)
```hcl
resource "github_repository_environment_deployment_policy" "branch_policies" {
  repository     = "my-app"
  environment    = "staging"
  branch_pattern = "release/*"
}

resource "github_repository_environment_deployment_policy" "tag_policies" {
  repository  = "my-app"
  environment = "staging"
  tag_pattern = "v*"
}
```

## Complete Example

```yaml
environments:
  - environment: production
    wait_timer: 300
    can_admins_bypass: false
    prevent_self_review: true
    reviewers:
      teams: ["platform-team"]
    deployment_policy:
      policy_type: protected_branches

  - environment: staging
    prevent_self_review: true
    deployment_policy:
      policy_type: selected_branches_and_tags
      branch_patterns: ["release/*", "main"]
      tag_patterns: ["v*", "rc-*"]

  - environment: development
    # No restrictions - any branch can deploy
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "reviewers: must be 6 or fewer" | Combined users + teams must be ≤ 6 |
| Custom policies not working | Ensure `policy_type: selected_branches_and_tags` |
| Deployment policies not created | Check `custom_branch_policies = true` in Terraform |

For more configuration options, see [DEVELOPERS_GUIDE.md](DEVELOPERS_GUIDE.md)
