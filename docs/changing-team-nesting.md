> [!IMPORTANT]
> Adding or removing a team's `parent` (making a top-level team nested, or a nested team top-level) is **not a normal config change**. Editing the YAML alone destroys and recreates the team on GitHub. Follow this runbook.
>
> Note: only **child ↔ root** transitions need this. Changing *which* parent a nested team has (parent A → parent B) is a normal in-place update — just edit the YAML.

## Why

Teams are managed by two resources so a child can reference its parent for correct create ordering (a single `for_each` self-reference cycles):

| Team has a `parent`? | Resource |
|---|---|
| no (top-level) | `github_team.team["<name>"]` |
| yes (nested) | `github_team.child_team["<name>"]` |

Adding a `parent` moves the team `github_team.team → github_team.child_team`; removing it moves it back. Terraform sees the old address disappear and a new one appear, so it plans a **destroy + create**. The destroy is real: **deleting a GitHub team removes its members and every repository grant.**

The fix: **move the state entry to the new address first, then apply** — so Terraform sees the same resource and does an in-place `parent_team_id` update instead.

## Steps

Prereq: an HCP Terraform token for the config repo's workspace (`state mv` makes no GitHub API calls — no App/PEM creds, no `-var-file`).

```bash
git clone https://github.com/G-Research/github-terraformer.git && cd github-terraformer/feature/github-repo-provisioning
ln -sf backend.tf.hcp backend.tf
export TF_CLOUD_ORGANIZATION=<TFC_ORG> TF_WORKSPACE=<WORKSPACE>   # config repo's tfc_org input and WORKSPACE variable
terraform init -input=false
terraform state pull > backup.tfstate   # always back up first
```

1. **Open a PR** editing `organisation/teams.yaml` — add or remove the team's `parent:` — and get it approved. Its first plan shows `1 to add, 1 to destroy` for the team (`github_team.child_team["<name>"]` + `github_team.team["<name>"]`) — **do not merge**.
2. **Move the state entry** to the new address (nothing changes on GitHub):
   ```bash
   # adding a parent (top-level -> nested):
   terraform state mv 'github_team.team["<name>"]' 'github_team.child_team["<name>"]'
   # removing a parent (nested -> top-level): reverse the two addresses
   ```
3. **Re-plan and merge.** The plan is now `0 to destroy` — a single in-place `parent_team_id` change. Merge; the apply just re-parents (or un-parents) the team, membership and repo grants intact.

> [!CAUTION]
> Between the `state mv` and the merge, state no longer matches `main`. Any other config PR merging in that window plans a destroy+create for this team. Approve first, then `state mv`, then merge immediately (freeze merges if the org is busy).

> [!NOTE]
> **One level of nesting only.** A team's `parent` must be a top-level team. Deleting a parent that still has children, or nesting under a nested team, fails the plan with `Error: Invalid index` — detach/reparent the children first.
