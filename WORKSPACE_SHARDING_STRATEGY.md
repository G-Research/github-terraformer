# Workspace Sharding Strategy for GitHub Terraformer

**Goal:** Reduce Terraform plan/apply time from 20 minutes to ~2-3 minutes per shard with parallel execution.

**Status:** Design Document
**Target:** 300+ repositories, ~1500 resources
**Approach:** Grouped sharding with 10 workspaces + 1 core workspace

---

## Table of Contents

1. [Strategy Overview](#strategy-overview)
2. [Sharding Schema](#sharding-schema)
3. [Architecture Changes](#architecture-changes)
4. [HCP Workspace Setup](#hcp-workspace-setup)
5. [Code Implementation](#code-implementation)
6. [Workflow Changes](#workflow-changes)
7. [Migration Plan](#migration-plan)
8. [Rollback Strategy](#rollback-strategy)

---

## Strategy Overview

### Current State
```
┌─────────────────────────────────────┐
│ Single Workspace                    │
│ github-configuration-prod-<org>-cli │
│                                     │
│ • 300+ repositories                 │
│ • ~1500 resources                   │
│ • Plan time: 10 minutes             │
│ • Apply time: 10 minutes            │
│ • Total: 20 minutes                 │
└─────────────────────────────────────┘
```

### Target State
```
┌────────────────────────────────────────────────────────────────┐
│ Core Workspace (shared resources)                             │
│ github-config-prod-<org>-core                                  │
│                                                                │
│ • Team ID mappings                                             │
│ • App ID mappings                                              │
│ • Plan time: ~30 seconds                                       │
└────────────────────────────────────────────────────────────────┘
                              ↓
        ┌────────────────────────────────────────────┐
        │     Parallel Shard Execution (10 shards)   │
        └────────────────────────────────────────────┘
                              ↓
    ┌───────┬───────┬───────┬───────┬───────┐
    │Shard 0│Shard 1│Shard 2│  ...  │Shard 9│
    │~30    │~30    │~30    │       │~30    │
    │repos  │repos  │repos  │       │repos  │
    │       │       │       │       │       │
    │2-3 min│2-3 min│2-3 min│       │2-3 min│
    └───────┴───────┴───────┴───────┴───────┘

Total time with parallel execution: ~3 minutes (vs 20 minutes)
Speedup: 6.7x improvement
```

---

## Sharding Schema

### Hash-Based Distribution

Use consistent hashing to distribute repositories evenly across shards:

```python
shard_number = hash(repo_name) % 10
```

This ensures:
- Even distribution (~30 repos per shard)
- Deterministic assignment (same repo always goes to same shard)
- No manual maintenance of shard assignments

### Shard Mapping

| Shard ID | Workspace Name | Est. Repos | Est. Resources | Plan Time |
|----------|----------------|------------|----------------|-----------|
| 0 | `github-config-prod-<org>-shard-0` | ~30 | ~150 | ~2-3 min |
| 1 | `github-config-prod-<org>-shard-1` | ~30 | ~150 | ~2-3 min |
| 2 | `github-config-prod-<org>-shard-2` | ~30 | ~150 | ~2-3 min |
| 3 | `github-config-prod-<org>-shard-3` | ~30 | ~150 | ~2-3 min |
| 4 | `github-config-prod-<org>-shard-4` | ~30 | ~150 | ~2-3 min |
| 5 | `github-config-prod-<org>-shard-5` | ~30 | ~150 | ~2-3 min |
| 6 | `github-config-prod-<org>-shard-6` | ~30 | ~150 | ~2-3 min |
| 7 | `github-config-prod-<org>-shard-7` | ~30 | ~150 | ~2-3 min |
| 8 | `github-config-prod-<org>-shard-8` | ~30 | ~150 | ~2-3 min |
| 9 | `github-config-prod-<org>-shard-9` | ~30 | ~150 | ~2-3 min |
| core | `github-config-prod-<org>-core` | n/a | ~50 | ~30 sec |

**Total Workspaces:** 11 (10 shards + 1 core)

---

## Architecture Changes

### Directory Structure

```
github-terraformer/
├── feature/
│   └── github-repo-provisioning/
│       ├── core/                          # NEW: Core workspace
│       │   ├── main.tf
│       │   ├── outputs.tf
│       │   ├── backend.tf.hcp
│       │   └── versions.tf
│       │
│       ├── shards/                        # NEW: Shard workspaces
│       │   ├── main.tf
│       │   ├── variables.tf
│       │   ├── backend.tf.hcp
│       │   ├── versions.tf
│       │   └── shard-calculator.sh        # Helper script
│       │
│       ├── modules/
│       │   └── terraform-github-repository/
│       │
│       ├── gcss_config/
│       │   ├── repos/                     # Existing repo configs
│       │   ├── importer_tmp_dir/          # Existing import configs
│       │   └── shard-mapping.json         # NEW: Cache of repo→shard mapping
│       │
│       ├── app-list.yaml
│       └── import-config.yaml
│
├── .github/
│   ├── workflows/
│   │   ├── tf-plan-sharded.yaml           # NEW: Sharded plan workflow
│   │   ├── tf-apply-sharded.yaml          # NEW: Sharded apply workflow
│   │   ├── tf-plan.yaml                   # KEEP: Fallback to monolithic
│   │   └── ...
│   │
│   └── actions/
│       ├── shard-calculator/              # NEW: Determine affected shards
│       │   └── action.yaml
│       └── ...
```

---

## HCP Workspace Setup

### Workspace Configuration

**Core Workspace:**
```
Name: github-config-prod-<org>-core
Type: CLI-driven
Terraform Version: Latest
Execution Mode: Remote
Auto Apply: False
```

**Variables (same for all workspaces):**
| Variable | Type | Value | Sensitive |
|----------|------|-------|-----------|
| `app_id` | Terraform | `<app-id>` | No |
| `app_installation_id` | Terraform | `<installation-id>` | No |
| `app_private_key` | Terraform | `<pem-content>` | Yes |
| `owner` | Terraform | `<org-name>` | No |
| `shard_id` | Terraform | `0-9` or `core` | No |

**Shard Workspaces (0-9):**
```
Name: github-config-prod-<org>-shard-{0-9}
Type: CLI-driven
Terraform Version: Latest
Execution Mode: Remote
Auto Apply: False
Parallelism: 15 (increased from default 10)
```

### Workspace Creation Script

Create all workspaces programmatically:

```bash
# scripts/create-workspaces.sh
#!/bin/bash
set -e

ORG="<your-org>"
TFC_ORG="<your-tfc-org>"
TFC_TOKEN="<your-token>"

# Create core workspace
tfe workspace create \
  --organization "$TFC_ORG" \
  --name "github-config-prod-${ORG}-core" \
  --execution-mode "remote" \
  --terraform-version "latest" \
  --auto-apply false

# Create shard workspaces
for i in {0..9}; do
  tfe workspace create \
    --organization "$TFC_ORG" \
    --name "github-config-prod-${ORG}-shard-${i}" \
    --execution-mode "remote" \
    --terraform-version "latest" \
    --auto-apply false
done
```

---

## Code Implementation

### 1. Core Workspace (`feature/github-repo-provisioning/core/main.tf`)

```hcl
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.9"
    }
  }
  cloud {
    organization = var.tfc_organization
    workspaces {
      name = "github-config-prod-${var.owner}-core"
    }
  }
}

provider "github" {
  owner = var.owner
  app_auth {
    id              = var.app_id
    installation_id = var.app_installation_id
    pem_file        = var.app_private_key
  }
}

# Load all team names from repo configs
locals {
  all_repos = merge(
    {
      for file_path in fileset(path.module, "../gcss_config/repos/*.yaml") :
      split(".yaml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "../gcss_config/repos/*.yml") :
      split(".yml", basename(file_path))[0] => yamldecode(file(file_path))
    }
  )

  # Extract all unique team names
  all_team_names = distinct(flatten([
    for repo, config in local.all_repos : concat(
      try(config.pull_teams, []),
      try(config.push_teams, []),
      try(config.admin_teams, []),
      try(config.maintain_teams, []),
      try(config.triage_teams, [])
    )
  ]))
}

# Fetch all team data once
data "github_team" "teams" {
  for_each = toset(local.all_team_names)
  slug     = each.value
}

# Export team ID mappings
output "team_ids" {
  description = "Mapping of team slugs to team IDs"
  value = {
    for slug, team in data.github_team.teams :
    slug => team.id
  }
}

# Load and export app mappings
locals {
  apps_map = {
    for app in yamldecode(file("../app-list.yaml")).apps :
    "app/${app.app_owner}/${app.app_slug}" => {
      app_id     = app.app_id
      app_slug   = app.app_slug
      app_owner  = app.app_owner
    }
  }
}

output "app_ids" {
  description = "Mapping of app keys to app IDs"
  value       = local.apps_map
}

# Fetch app node IDs for apps referenced in branch protections
locals {
  # Find all app references in branch protections
  app_slugs = distinct(flatten([
    for repo, config in local.all_repos : [
      for bp in try(config.branch_protections_v4, []) : [
        for actor in concat(
          try(bp.force_push_bypassers, []),
          try(bp.push_restrictions, []),
          try(bp.required_pull_request_reviews.dismissal_restrictions, []),
          try(bp.required_pull_request_reviews.pull_request_bypassers, [])
        ) : split("/", actor)[1]
        if startswith(actor, "app/")
      ]
    ]
  ]))
}

data "github_app" "apps" {
  for_each = toset(local.app_slugs)
  slug     = each.value
}

output "app_node_ids" {
  description = "Mapping of app slugs to node IDs"
  value = {
    for slug, app in data.github_app.apps :
    slug => app.node_id
  }
}
```

### 2. Shard Workspace (`feature/github-repo-provisioning/shards/main.tf`)

```hcl
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.9"
    }
  }
  cloud {
    organization = var.tfc_organization
    workspaces {
      name = "github-config-prod-${var.owner}-shard-${var.shard_id}"
    }
  }
}

provider "github" {
  owner = var.owner
  app_auth {
    id              = var.app_id
    installation_id = var.app_installation_id
    pem_file        = var.app_private_key
  }
}

# Import core workspace outputs
data "terraform_remote_state" "core" {
  backend = "remote"
  config = {
    organization = var.tfc_organization
    workspaces = {
      name = "github-config-prod-${var.owner}-core"
    }
  }
}

locals {
  team_ids     = data.terraform_remote_state.core.outputs.team_ids
  app_ids      = data.terraform_remote_state.core.outputs.app_ids
  app_node_ids = data.terraform_remote_state.core.outputs.app_node_ids
}

# Load all repos
locals {
  all_repos_raw = merge(
    {
      for file_path in fileset(path.module, "../gcss_config/repos/*.yaml") :
      split(".yaml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "../gcss_config/repos/*.yml") :
      split(".yml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "../gcss_config/importer_tmp_dir/*.yaml") :
      split(".yaml", basename(file_path))[0] => yamldecode(file(file_path))
    },
    {
      for file_path in fileset(path.module, "../gcss_config/importer_tmp_dir/*.yml") :
      split(".yml", basename(file_path))[0] => yamldecode(file(file_path))
    }
  )

  # Filter repos for this shard using hash-based distribution
  shard_repos = {
    for name, config in local.all_repos_raw :
    name => config
    if parseint(sha256(name), 16) % 10 == var.shard_id
  }

  # Separate generated vs new repos within this shard
  generated_repos = {
    for name, config in local.shard_repos :
    name => config
    if contains(keys(config), "id")  # Generated repos have IDs
  }

  new_repos = {
    for name, config in local.shard_repos :
    name => config
    if !contains(keys(config), "id")  # New repos don't have IDs
  }
}

# Import blocks for generated repos (same as before, but scoped to shard)
import {
  for_each = local.generated_repos
  to       = module.repository[each.key].github_repository.repository
  id       = each.key
}

import {
  for_each = local.generated_repos
  to       = module.repository[each.key].github_branch_default.default[0]
  id       = each.key
}

# Branch protections import (same pattern, scoped to shard)
locals {
  flattened_generated_branch_protections_v4 = flatten([
    for repo, config in local.generated_repos : [
      for branch_protection in try(config.branch_protections_v4, []) : {
        repository        = repo
        branch_protection = branch_protection
      }
    ]
  ])
}

import {
  for_each = {
    for item in local.flattened_generated_branch_protections_v4 :
    "${item.repository}:${item.branch_protection.pattern}" => item
  }

  to = module.repository[each.value.repository].github_branch_protection.branch_protection[each.value.branch_protection.pattern]
  id = format("%s:%s", each.value.repository, each.value.branch_protection.pattern)
}

# Collaborators import (scoped to shard)
locals {
  all_generated_collaborators = {
    for repo, config in local.generated_repos : repo => concat(
      try([for i in config.pull_collaborators : {username : i, permission = "pull"}], []),
      try([for i in config.push_collaborators : {username : i, permission = "push"}], []),
      try([for i in config.admin_collaborators : {username : i, permission = "admin"}], []),
      try([for i in config.maintain_collaborators : {username : i, permission = "maintain"}], []),
      try([for i in config.triage_collaborators : {username : i, permission = "triage"}], [])
    )
  }
}

import {
  for_each = toset(flatten([
    for repo, collaborators in local.all_generated_collaborators : [
      for collaborator in collaborators :
      "${repo}:${collaborator.username}"
    ]
  ]))

  to = module.repository[split(":", each.value)[0]].github_repository_collaborator.collaborator[split(":", each.value)[1]]
  id = each.value
}

# Teams import (using cached team IDs from core)
locals {
  all_generated_teams = {
    for repo, config in local.generated_repos : repo => concat(
      try([for i in config.pull_teams : {name : i, permission = "pull"}], []),
      try([for i in config.push_teams : {name : i, permission = "push"}], []),
      try([for i in config.admin_teams : {name : i, permission = "admin"}], []),
      try([for i in config.maintain_teams : {name : i, permission = "maintain"}], []),
      try([for i in config.triage_teams : {name : i, permission = "triage"}], [])
    )
  }
}

import {
  for_each = toset(flatten([
    for repo, teams in local.all_generated_teams : [
      for team in teams :
      "${local.team_ids[team.name]}:${repo}"
    ]
  ]))

  to = module.repository[split(":", each.value)[1]].github_team_repository.team_repository_by_slug[split(":", each.value)[0]]
  id = each.value
}

# Repository module (same as before, but with cached team/app IDs)
module "repository" {
  source   = "../modules/terraform-github-repository"
  for_each = merge(local.generated_repos, local.new_repos)

  name                   = each.key
  default_branch         = try(each.value.default_branch, "")
  description            = try(each.value.description, "")
  visibility             = try(each.value.visibility, "private")
  # ... all other repository settings ...

  # Use cached team IDs instead of data sources
  pull_teams     = try([for i in each.value.pull_teams : local.team_ids[i]], [])
  push_teams     = try([for i in each.value.push_teams : local.team_ids[i]], [])
  admin_teams    = try([for i in each.value.admin_teams : local.team_ids[i]], [])
  maintain_teams = try([for i in each.value.maintain_teams : local.team_ids[i]], [])
  triage_teams   = try([for i in each.value.triage_teams : local.team_ids[i]], [])

  # Branch protections with cached app node IDs
  branch_protections_v4 = try([
    for bp in try(each.value.branch_protections_v4, []) : {
      pattern              = bp.pattern
      allows_deletions     = try(bp.allows_deletions, false)
      allows_force_pushes  = try(bp.allows_force_pushes, false)
      force_push_bypassers = try([
        for bypasser in bp.force_push_bypassers :
        !startswith(bypasser, "app/") ? bypasser : local.app_node_ids[split("/", bypasser)[1]]
      ], [])
      # ... rest of branch protection config ...
    }
  ], [])

  # ... rest of module configuration ...
}

# Rulesets (same pattern as before, scoped to shard)
locals {
  shard_rulesets_flattened = flatten([
    for repo, config in merge(local.generated_repos, local.new_repos) : [
      for ruleset in try(config.rulesets, []) : {
        repository = repo
        ruleset    = ruleset
      }
    ]
  ])

  shard_rulesets_map = {
    for idx, item in local.shard_rulesets_flattened :
    sha256("${item.repository}/${item.ruleset.name}") => item
  }

  generated_rulesets_map = {
    for key, item in local.shard_rulesets_map :
    key => item
    if contains(keys(local.generated_repos), item.repository)
  }
}

import {
  for_each = local.generated_rulesets_map
  to       = github_repository_ruleset.ruleset[each.key]
  id       = format("%s:%s", each.value.repository, each.value.ruleset.id)
}

resource "github_repository_ruleset" "ruleset" {
  depends_on = [module.repository]

  for_each   = local.shard_rulesets_map
  name       = each.value.ruleset.name
  enforcement = each.value.ruleset.enforcement
  target     = each.value.ruleset.target
  repository = each.value.repository

  # ... rest of ruleset configuration using cached app IDs ...

  dynamic "bypass_actors" {
    for_each = try(each.value.ruleset.bypass_actors, [])

    content {
      actor_id = (
        startswith(bypass_actors.value.name, "team/")
        ? local.team_ids[replace(bypass_actors.value.name, "team/", "")]
        : (
          startswith(bypass_actors.value.name, "app/")
          ? local.app_ids[bypass_actors.value.name].app_id
          : local.ruleset_actors[bypass_actors.value.name].actor_id
        )
      )
      # ... rest of bypass_actors config ...
    }
  }
}

locals {
  ruleset_actors = {
    "repository-admin-role"   = { actor_type = "RepositoryRole", actor_id = 5 }
    "organization-admin-role" = { actor_type = "OrganizationAdmin", actor_id = 1 }
    "maintain-role"           = { actor_type = "RepositoryRole", actor_id = 2 }
    "write-role"              = { actor_type = "RepositoryRole", actor_id = 4 }
  }
}
```

### 3. Variables (`feature/github-repo-provisioning/shards/variables.tf`)

```hcl
variable "owner" {
  description = "GitHub organization name"
  type        = string
}

variable "app_id" {
  description = "GitHub App ID"
  type        = string
}

variable "app_installation_id" {
  description = "GitHub App Installation ID"
  type        = string
}

variable "app_private_key" {
  description = "GitHub App private key (PEM format)"
  type        = string
  sensitive   = true
}

variable "shard_id" {
  description = "Shard ID (0-9)"
  type        = number
  validation {
    condition     = var.shard_id >= 0 && var.shard_id <= 9
    error_message = "Shard ID must be between 0 and 9"
  }
}

variable "tfc_organization" {
  description = "Terraform Cloud organization name"
  type        = string
}
```

### 4. Shard Calculator Script (`feature/github-repo-provisioning/shards/shard-calculator.sh`)

```bash
#!/bin/bash
# Calculate which shard a repository belongs to

REPO_NAME="$1"

if [ -z "$REPO_NAME" ]; then
  echo "Usage: $0 <repo-name>"
  exit 1
fi

# Calculate shard using SHA256 hash (matching Terraform logic)
HASH=$(echo -n "$REPO_NAME" | sha256sum | awk '{print $1}')
# Convert first 16 hex chars to decimal and mod 10
SHARD=$((0x${HASH:0:16} % 10))

echo "$SHARD"
```

---

## Workflow Changes

### 1. Shard Calculator Action (`.github/actions/shard-calculator/action.yaml`)

```yaml
name: "Calculate Affected Shards"
description: "Determines which workspace shards are affected by file changes"
inputs:
  base-ref:
    description: "Base git ref for comparison"
    required: false
    default: "origin/main"
  head-ref:
    description: "Head git ref for comparison"
    required: false
    default: "HEAD"
outputs:
  affected-shards:
    description: "JSON array of affected shard IDs"
    value: ${{ steps.calculate.outputs.shards }}
  run-core:
    description: "Whether core workspace needs to run"
    value: ${{ steps.calculate.outputs.run_core }}
  changed-repos:
    description: "List of changed repository names"
    value: ${{ steps.calculate.outputs.repos }}

runs:
  using: "composite"
  steps:
    - name: Calculate affected shards
      id: calculate
      shell: bash
      run: |
        set -e

        # Get changed files
        CHANGED_FILES=$(git diff --name-only ${{ inputs.base-ref }}...${{ inputs.head-ref }})

        # Check if app-list.yaml or core files changed
        RUN_CORE="false"
        if echo "$CHANGED_FILES" | grep -qE "(app-list\.yaml|import-config\.yaml|core/)"; then
          RUN_CORE="true"
        fi
        echo "run_core=$RUN_CORE" >> $GITHUB_OUTPUT

        # Extract changed repo YAML files
        CHANGED_REPOS=$(echo "$CHANGED_FILES" \
          | grep -E "gcss_config/(repos|importer_tmp_dir)/.*\.(yaml|yml)$" \
          | sed 's|gcss_config/repos/||; s|gcss_config/importer_tmp_dir/||; s|\.ya?ml$||' \
          | sort -u)

        if [ -z "$CHANGED_REPOS" ]; then
          echo "No repository configs changed"
          echo "shards=[]" >> $GITHUB_OUTPUT
          echo "repos=" >> $GITHUB_OUTPUT
          exit 0
        fi

        # Calculate shard for each changed repo
        SHARDS=()
        for REPO in $CHANGED_REPOS; do
          # Calculate shard using SHA256 (matching Terraform logic)
          HASH=$(echo -n "$REPO" | sha256sum | awk '{print $1}')
          SHARD=$((0x${HASH:0:16} % 10))
          SHARDS+=($SHARD)
        done

        # Get unique shard IDs
        UNIQUE_SHARDS=$(printf '%s\n' "${SHARDS[@]}" | sort -u | jq -R . | jq -s .)

        echo "shards=$UNIQUE_SHARDS" >> $GITHUB_OUTPUT
        echo "repos=$(echo "$CHANGED_REPOS" | tr '\n' ',' | sed 's/,$//')" >> $GITHUB_OUTPUT

        echo "Changed repos: $CHANGED_REPOS"
        echo "Affected shards: $UNIQUE_SHARDS"
```

### 2. Sharded Plan Workflow (`.github/workflows/tf-plan-sharded.yaml`)

```yaml
name: Terraform Plan (Sharded)

on:
  workflow_call:
    inputs:
      commit_sha:
        type: string
        description: 'The commit SHA to plan'
        required: true
      gcss_ref:
        type: string
        description: "GCSS ref to checkout"
        required: false
        default: "main"
      tfc_org:
        type: string
        description: 'The Terraform Cloud organization'
        required: true
    secrets:
      app_private_key:
        required: true
      gh_token:
        required: true
      tfc_token:
        required: true

jobs:
  calculate-shards:
    runs-on: ubuntu-latest
    outputs:
      affected-shards: ${{ steps.shards.outputs.shards }}
      run-core: ${{ steps.shards.outputs.run_core }}
      changed-repos: ${{ steps.shards.outputs.repos }}
    steps:
      - name: Checkout config repo
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ inputs.commit_sha }}
          token: ${{ secrets.gh_token }}

      - name: Calculate affected shards
        id: shards
        uses: ./.github/actions/shard-calculator
        with:
          base-ref: origin/main
          head-ref: ${{ inputs.commit_sha }}

  # Core workspace plan (runs if core files changed)
  terraform-plan-core:
    needs: calculate-shards
    if: needs.calculate-shards.outputs.run-core == 'true'
    runs-on: ubuntu-latest
    environment: plan
    steps:
      - name: Checkout GCSS
        uses: actions/checkout@v4
        with:
          repository: G-Research/github-terraformer
          ref: ${{ inputs.gcss_ref }}
          persist-credentials: false

      - name: GCSS config setup
        uses: ./.github/actions/gcss-config-setup
        with:
          checkout-sha: ${{ inputs.commit_sha }}
          checkout-token: ${{ secrets.gh_token }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          cli_config_credentials_token: ${{ secrets.tfc_token }}

      - name: Terraform Init (Core)
        working-directory: feature/github-repo-provisioning/core
        run: terraform init -input=false
        env:
          TF_CLOUD_ORGANIZATION: ${{ inputs.tfc_org }}
          TF_WORKSPACE: github-config-prod-${{ vars.WORKSPACE }}-core

      - name: Terraform Plan (Core)
        working-directory: feature/github-repo-provisioning/core
        run: |
          terraform plan -no-color -input=false -out=tfplan
          terraform show -json tfplan > tfplan.json

      - name: Upload Core Plan
        uses: actions/upload-artifact@v4
        with:
          name: core-plan
          path: feature/github-repo-provisioning/core/tfplan.json

  # Shard workspaces plan (runs in parallel for affected shards)
  terraform-plan-shards:
    needs: calculate-shards
    if: needs.calculate-shards.outputs.affected-shards != '[]'
    runs-on: ubuntu-latest
    environment: plan
    strategy:
      matrix:
        shard_id: ${{ fromJSON(needs.calculate-shards.outputs.affected-shards) }}
      max-parallel: 10  # Run all shards in parallel
      fail-fast: false  # Continue even if one shard fails
    permissions:
      pull-requests: write
      contents: read
    steps:
      - name: Generate a token
        uses: actions/create-github-app-token@v2
        id: generate-token
        with:
          app-id: ${{ vars.APP_ID }}
          private-key: ${{ secrets.app_private_key }}
          owner: ${{ github.repository_owner }}

      - name: Create in-progress check-run
        uses: actions/github-script@v7
        env:
          COMMIT_SHA: ${{ inputs.commit_sha }}
          SHARD_ID: ${{ matrix.shard_id }}
        with:
          github-token: ${{ steps.generate-token.outputs.token }}
          script: |
            const detailsUrl = `${context.serverUrl}/${context.payload.repository.full_name}/actions/runs/${context.runId}`;

            await github.rest.checks.create({
              owner: context.payload.repository.owner.login,
              repo: context.payload.repository.name,
              name: `Terraform plan (shard ${process.env.SHARD_ID})`,
              head_sha: process.env.COMMIT_SHA,
              status: "in_progress",
              details_url: detailsUrl,
              output: {
                title: `Terraform Plan running (shard ${process.env.SHARD_ID})`,
                summary: `Follow workflow logs for details: ${detailsUrl}`
              }
            });

      - name: Checkout GCSS
        uses: actions/checkout@v4
        with:
          repository: G-Research/github-terraformer
          ref: ${{ inputs.gcss_ref }}
          persist-credentials: false

      - name: GCSS config setup
        uses: ./.github/actions/gcss-config-setup
        with:
          checkout-sha: ${{ inputs.commit_sha }}
          checkout-token: ${{ secrets.gh_token }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          cli_config_credentials_token: ${{ secrets.tfc_token }}

      - name: Terraform Init (Shard ${{ matrix.shard_id }})
        working-directory: feature/github-repo-provisioning/shards
        run: terraform init -input=false
        env:
          TF_CLOUD_ORGANIZATION: ${{ inputs.tfc_org }}
          TF_WORKSPACE: github-config-prod-${{ vars.WORKSPACE }}-shard-${{ matrix.shard_id }}

      - name: Terraform Plan (Shard ${{ matrix.shard_id }})
        working-directory: feature/github-repo-provisioning/shards
        run: |
          terraform plan -no-color -input=false -out=tfplan \
            -var="shard_id=${{ matrix.shard_id }}"
          terraform show -json tfplan > tfplan-shard-${{ matrix.shard_id }}.json

      - name: Parse Plan Output
        id: parse
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const planPath = `feature/github-repo-provisioning/shards/tfplan-shard-${{ matrix.shard_id }}.json`;
            const planJson = JSON.parse(fs.readFileSync(planPath, 'utf8'));

            const changes = {
              create: [],
              update: [],
              delete: [],
              import: [],
              recreate: []
            };

            // Parse resource changes (same logic as original)
            if (planJson.resource_changes) {
              for (const rc of planJson.resource_changes) {
                const actions = rc.change.actions;
                if (actions.includes('create')) changes.create.push(rc.address);
                else if (actions.includes('update')) changes.update.push(rc.address);
                else if (actions.includes('delete')) changes.delete.push(rc.address);
                // ... handle import and recreate
              }
            }

            core.setOutput('changes', JSON.stringify(changes));

            const totalChanges =
              changes.create.length +
              changes.update.length +
              changes.delete.length +
              changes.recreate.length;

            core.setOutput('has_changes', totalChanges > 0 ? 'true' : 'false');
            return changes;

      - name: Update check-run with results
        if: always()
        uses: actions/github-script@v7
        env:
          COMMIT_SHA: ${{ inputs.commit_sha }}
          SHARD_ID: ${{ matrix.shard_id }}
          HAS_CHANGES: ${{ steps.parse.outputs.has_changes }}
          CHANGES: ${{ steps.parse.outputs.changes }}
        with:
          github-token: ${{ steps.generate-token.outputs.token }}
          script: |
            const detailsUrl = `${context.serverUrl}/${context.payload.repository.full_name}/actions/runs/${context.runId}`;
            const changes = JSON.parse(process.env.CHANGES);
            const hasChanges = process.env.HAS_CHANGES === 'true';

            const title = hasChanges
              ? `Terraform Plan Succeeded (shard ${process.env.SHARD_ID}, with changes)`
              : `Terraform Plan Succeeded (shard ${process.env.SHARD_ID}, no changes)`;

            const summary = hasChanges
              ? `Shard ${process.env.SHARD_ID}: ${changes.create.length} to add, ${changes.update.length} to change, ${changes.delete.length} to destroy`
              : `Shard ${process.env.SHARD_ID}: No changes`;

            await github.rest.checks.create({
              owner: context.payload.repository.owner.login,
              repo: context.payload.repository.name,
              name: `Terraform plan (shard ${process.env.SHARD_ID})`,
              head_sha: process.env.COMMIT_SHA,
              status: "completed",
              conclusion: "success",
              details_url: detailsUrl,
              output: {
                title: title,
                summary: summary
              }
            });

      - name: Upload Shard Plan
        uses: actions/upload-artifact@v4
        with:
          name: shard-${{ matrix.shard_id }}-plan
          path: feature/github-repo-provisioning/shards/tfplan-shard-${{ matrix.shard_id }}.json

  # Summary job (aggregates results from all shards)
  plan-summary:
    needs: [calculate-shards, terraform-plan-core, terraform-plan-shards]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - name: Create overall summary
        uses: actions/github-script@v7
        env:
          CHANGED_REPOS: ${{ needs.calculate-shards.outputs.changed-repos }}
          AFFECTED_SHARDS: ${{ needs.calculate-shards.outputs.affected-shards }}
        with:
          script: |
            const shards = JSON.parse(process.env.AFFECTED_SHARDS || '[]');
            const repos = process.env.CHANGED_REPOS || 'none';

            core.summary
              .addHeading('Terraform Plan Summary (Sharded)')
              .addTable([
                [{data: 'Metric', header: true}, {data: 'Value', header: true}],
                ['Changed Repositories', repos],
                ['Affected Shards', shards.join(', ') || 'none'],
                ['Total Shards Planned', shards.length.toString()],
              ])
              .write();
```

### 3. Sharded Apply Workflow (`.github/workflows/tf-apply-sharded.yaml`)

Similar structure to plan workflow, but with apply steps:

```yaml
name: Terraform Apply (Sharded)

on:
  workflow_call:
    inputs:
      commit_sha:
        type: string
        required: true
      gcss_ref:
        type: string
        required: false
        default: "main"
      tfc_org:
        type: string
        required: true
    secrets:
      app_private_key:
        required: true
      gh_token:
        required: true
      tfc_token:
        required: true

jobs:
  calculate-shards:
    # Same as plan workflow
    runs-on: ubuntu-latest
    outputs:
      affected-shards: ${{ steps.shards.outputs.shards }}
      run-core: ${{ steps.shards.outputs.run_core }}
    steps:
      - name: Checkout config repo
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ inputs.commit_sha }}
          token: ${{ secrets.gh_token }}

      - name: Calculate affected shards
        id: shards
        uses: ./.github/actions/shard-calculator
        with:
          base-ref: origin/main
          head-ref: ${{ inputs.commit_sha }}

  # Apply core first (dependency for shards)
  terraform-apply-core:
    needs: calculate-shards
    if: needs.calculate-shards.outputs.run-core == 'true'
    runs-on: ubuntu-latest
    environment: promote
    steps:
      # Similar to plan, but with terraform apply
      - name: Checkout GCSS
        uses: actions/checkout@v4
        with:
          repository: G-Research/github-terraformer
          ref: ${{ inputs.gcss_ref }}
          persist-credentials: false

      - name: GCSS config setup
        uses: ./.github/actions/gcss-config-setup
        with:
          checkout-sha: ${{ inputs.commit_sha }}
          checkout-token: ${{ secrets.gh_token }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          cli_config_credentials_token: ${{ secrets.tfc_token }}

      - name: Terraform Init (Core)
        working-directory: feature/github-repo-provisioning/core
        run: terraform init -input=false
        env:
          TF_CLOUD_ORGANIZATION: ${{ inputs.tfc_org }}
          TF_WORKSPACE: github-config-prod-${{ vars.WORKSPACE }}-core

      - name: Terraform Apply (Core)
        working-directory: feature/github-repo-provisioning/core
        run: terraform apply -auto-approve -input=false

  # Apply shards in parallel (after core completes)
  terraform-apply-shards:
    needs: [calculate-shards, terraform-apply-core]
    if: |
      always() &&
      needs.calculate-shards.outputs.affected-shards != '[]' &&
      (needs.terraform-apply-core.result == 'success' || needs.terraform-apply-core.result == 'skipped')
    runs-on: ubuntu-latest
    environment: promote
    strategy:
      matrix:
        shard_id: ${{ fromJSON(needs.calculate-shards.outputs.affected-shards) }}
      max-parallel: 10
      fail-fast: false
    steps:
      - name: Checkout GCSS
        uses: actions/checkout@v4
        with:
          repository: G-Research/github-terraformer
          ref: ${{ inputs.gcss_ref }}
          persist-credentials: false

      - name: GCSS config setup
        uses: ./.github/actions/gcss-config-setup
        with:
          checkout-sha: ${{ inputs.commit_sha }}
          checkout-token: ${{ secrets.gh_token }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          cli_config_credentials_token: ${{ secrets.tfc_token }}

      - name: Terraform Init (Shard ${{ matrix.shard_id }})
        working-directory: feature/github-repo-provisioning/shards
        run: terraform init -input=false
        env:
          TF_CLOUD_ORGANIZATION: ${{ inputs.tfc_org }}
          TF_WORKSPACE: github-config-prod-${{ vars.WORKSPACE }}-shard-${{ matrix.shard_id }}

      - name: Terraform Apply (Shard ${{ matrix.shard_id }})
        working-directory: feature/github-repo-provisioning/shards
        run: |
          terraform apply -auto-approve -input=false \
            -var="shard_id=${{ matrix.shard_id }}"
```

---

## Migration Plan

### Phase 1: Setup (Week 1)

**Day 1-2: Workspace Creation**
1. Create HCP workspaces (core + 10 shards)
2. Configure variables for all workspaces
3. Set up access permissions

**Day 3-4: Code Implementation**
1. Create `core/` directory structure
2. Create `shards/` directory structure
3. Implement shard calculator action
4. Test locally with `terraform init` in each directory

**Day 5: Workflow Creation**
1. Create sharded plan workflow
2. Create sharded apply workflow
3. Create shard calculator action

### Phase 2: Testing (Week 2)

**Test on a separate branch:**

```bash
# Create test branch
git checkout -b test-sharding

# Test core workspace
cd feature/github-repo-provisioning/core
terraform init
terraform plan

# Test shard 0
cd ../shards
terraform init
terraform plan -var="shard_id=0"

# Test shard 1
terraform plan -var="shard_id=1"
```

**Validation checklist:**
- [ ] Core workspace outputs team_ids correctly
- [ ] Core workspace outputs app_ids correctly
- [ ] Shard 0 can read core outputs
- [ ] Shard 0 filters repos correctly (check count)
- [ ] Shard 1 filters repos correctly (check count)
- [ ] No repo appears in multiple shards
- [ ] Shard calculator script works
- [ ] Shard calculator action works
- [ ] All 10 shards sum to ~300 repos

### Phase 3: Migration (Week 3)

**State Migration Strategy:**

Option A: **Import-based migration** (Safer, recommended)
- Keep monolithic workspace as backup
- Let sharded workspaces import resources
- Compare plans between old and new
- Switch workflows after validation

Option B: **State migration** (Faster but riskier)
- Use `terraform state mv` to move resources
- Requires careful state manipulation
- Higher risk of errors

**Recommended: Option A**

1. **Deploy sharded infrastructure alongside monolithic:**
   ```bash
   # Monolithic workspace still exists: github-config-prod-<org>-cli
   # New workspaces created: github-config-prod-<org>-{core,shard-0..9}
   ```

2. **Run parallel plans:**
   ```bash
   # Plan in monolithic (should show no changes)
   terraform plan -chdir=feature/github-repo-provisioning

   # Plan in all shards (should show imports only)
   for i in {0..9}; do
     terraform plan -chdir=feature/github-repo-provisioning/shards \
       -var="shard_id=$i"
   done
   ```

3. **Validate identical state:**
   - Export resources from monolithic workspace
   - Export resources from all sharded workspaces
   - Compare: should be identical

4. **Switch workflows:**
   - Update main workflow to use sharded version
   - Keep monolithic as backup

5. **Decommission old workspace:**
   - After 1 week of successful sharded operations
   - Archive monolithic workspace (don't delete)

### Phase 4: Optimization (Week 4)

**Fine-tuning:**
1. Adjust shard count if needed (8-12 shards)
2. Optimize parallel execution (max-parallel setting)
3. Monitor HCP Terraform costs
4. Add caching for team/app lookups
5. Create documentation

---

## Rollback Strategy

If issues arise, you can quickly rollback:

### Immediate Rollback (Minutes)

1. **Revert workflow changes:**
   ```bash
   git revert <sharding-commit>
   git push
   ```

2. **Workflows automatically use monolithic workspace**

### Full Rollback (If state diverged)

1. **Stop all shard operations**
2. **Export state from sharded workspaces:**
   ```bash
   terraform state pull > shard-0-state.json
   # Repeat for all shards
   ```

3. **Merge states back to monolithic** (if needed)
4. **Resume with monolithic workspace**

---

## Success Metrics

Track these metrics to validate the sharding strategy:

| Metric | Before | Target | Measurement |
|--------|--------|--------|-------------|
| Plan time (PR) | 10 min | 3 min | Workflow duration |
| Apply time (merge) | 10 min | 3 min | Workflow duration |
| Feedback time | 20 min | 3-5 min | PR to green check |
| Failed runs | Low | Same/Better | Error rate |
| State drift incidents | Low | Same/Better | Drift check results |
| HCP Terraform costs | Baseline | <1.5x | Monthly bill |

---

## Cost Considerations

**HCP Terraform Pricing Impact:**

Current: 1 workspace × 20 min/run × N runs/month
Sharded: 11 workspaces × 3 min/run × N runs/month

**Estimated cost change:**
- Workspace count: 11x more
- Run time per workspace: ~7x less
- Total compute time: ~1.5x more (but distributed)
- Monthly cost increase: ~50% (varies by HCP plan)

**Optimization:**
- Only run affected shards (not all 10)
- Core workspace runs rarely
- Typical PR affects 1-3 shards only

---

## Next Steps

1. **Review this strategy** with your team
2. **Get approval** for HCP workspace creation
3. **Schedule implementation** (3-4 weeks)
4. **Assign tasks:**
   - DevOps: HCP workspace setup
   - Terraform: Code implementation
   - Platform: Workflow updates
   - QA: Testing and validation

5. **Start with Phase 1** when ready

---

## Questions & Considerations

**Q: Why 10 shards instead of more/less?**
A: 10 shards balances:
- Manageable workspace count
- Significant speedup (6-7x)
- Even distribution (~30 repos per shard)
- Can adjust to 8-12 if needed

**Q: What if a repo needs to move shards?**
A: Repos are hash-assigned, so they never move. The hash is deterministic based on repo name.

**Q: What about shared resources like teams?**
A: Core workspace manages shared data sources, shards import via `terraform_remote_state`.

**Q: Can we shard more granularly (per-repo)?**
A: Yes, but adds complexity. Start with 10 shards, then evaluate if per-repo is needed.

**Q: What about import workflow?**
A: Import workflow needs updates to:
1. Calculate target shard for new repo
2. Place YAML in correct directory
3. Trigger plan/apply for that specific shard

**Q: Backwards compatibility?**
A: Monolithic workspace remains as fallback. Workflows can switch between modes via feature flag.

---

## Support & Maintenance

**After implementation:**
- Monitor shard distribution (should be ~30 repos each)
- Watch for outlier shards (significantly more/less repos)
- Consider rebalancing if org grows to 500+ repos
- Update this document with lessons learned

**Documentation:**
- Add runbook for common operations
- Document troubleshooting steps
- Create team training materials

---

**End of Strategy Document**
