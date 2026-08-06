> [!IMPORTANT]
> Renaming a team or changing a member's username is **not a normal config change**. Editing the YAML on its own destroys and recreates the resource. Follow this runbook.

## 🔑 Why a rename is different

Organisation resources are created with `for_each`, and the map key is taken straight from the YAML. In Terraform, that key **is** the resource's identity:

| Resource | `for_each` key | Defined in |
|---|---|---|
| `github_team.team` | team `name` | `feature/github-repo-provisioning/teams.tf` |
| `github_membership.member` | `username` | `feature/github-repo-provisioning/members.tf` |
| `github_team_membership.membership` | `"<username>/<team name>"` | `feature/github-repo-provisioning/members.tf` |
| `github_repository_collaborator.collaborator` | `username` | `modules/terraform-github-repository/main.tf` |

Change the `name` in `organisation/teams.yaml` and Terraform does not see a rename — it sees `github_team.team["Old"]` disappear and `github_team.team["New"]` appear, and plans a **destroy + create**. The destroy is real: deleting a GitHub team removes its membership and every repository grant that references it. The equivalent mistake on a member destroys their `github_membership`, which removes them from the organisation entirely and turns their re-add into a pending invitation.

The fix is always the same shape: **move the state entry to the new key first, then change the YAML**, so the apply has nothing to destroy.

> [!NOTE]
> `moved {}` blocks are not an option here. They are static HCL and would have to live in `feature/github-repo-provisioning/*.tf` in this repository, but renames are driven by YAML in the config repo — a config-repo PR cannot carry one. State operations are used instead.

## 🧰 Before you start

- **Who runs this**: someone with an HCP Terraform token for the config repo's workspace. `terraform state mv` and `terraform state rm` are local operations on remote state — they make **no GitHub API calls**, so no GitHub App credentials or PEM key are needed, and no `-var-file` is required.
- **When**: while no plan or apply is in flight. State operations take the workspace lock, and any plan file saved before the move is stale and must be re-planned.

> [!CAUTION]
> The moment you touch state, it stops matching the YAML on `main` — the state says `New`, the merged config still says `Old`. **Any other PR that merges in that window applies a destroy.** Get the rename PR reviewed and approved *first*, then do the state operation, then merge straight away. Ask for a merge freeze on the config repo if the org is busy.

Set up a working copy of this repository pointed at the live workspace:

```bash
git clone https://github.com/G-Research/github-terraformer.git && cd github-terraformer/feature/github-repo-provisioning
```

```bash
ln -sf backend.tf.hcp backend.tf
```

```bash
export TF_CLOUD_ORGANIZATION=<TFC_ORG> TF_WORKSPACE=<WORKSPACE>   # the config repo's tfc_org input and WORKSPACE variable
terraform init -input=false
```

Always take a backup before touching state:

```bash
terraform state pull > backup.tfstate
```

If a move goes wrong, roll the state back from the version history in the HCP Terraform UI, or run the `terraform state mv` in reverse.

## 🏷️ Renaming a team

A team rename is **performed by the apply** — GCSS renames the team on GitHub. Renaming changes the team's slug, and `repos/*.yaml` references teams **by slug**, so this takes **two PRs** in a fixed order.

Throughout, `Old Name` / `New Name` are the values of `name` in `organisation/teams.yaml`, and `old-slug` / `new-slug` are their GitHub slugs (lower-cased, non-alphanumerics replaced with `-`).

### 1. Open both PRs and get them approved

Prepare them now; merge neither yet.

**PR 1 — organisation files only.** Do **not** touch `repos/*.yaml` here.

- `organisation/teams.yaml` — change `name`, and update `slug` to the new slug. `slug` is importer-owned and is only read by the bootstrap `import {}` blocks, but a stale value breaks a future re-import.
- `organisation/members.yaml` — change every `teams[].name` that references the team. `validate-org` requires an exact-case match against `teams.yaml`, so this has to be in the same PR.

**PR 2 — repository files only.** See step 5 for its contents.

> [!NOTE]
> PR 1's first plan **will** show a destroy — the state has not moved yet. Expect `2 to add, 0 to change, 2 to destroy` with a `Resources to recreate:` section. Do not merge it; step 4 re-plans it clean.

### 2. Find every affected state address

```bash
terraform state list | grep -E 'github_team\.team|github_team_membership\.membership'
```

You are looking for `github_team.team["Old Name"]` plus one `github_team_membership.membership["<username>/Old Name"]` for every member who lists the team in `organisation/members.yaml`.

### 3. Move them to the new key

```bash
terraform state mv 'github_team.team["Old Name"]' 'github_team.team["New Name"]'
```

Then one move per membership — the key contains a `/` and may contain spaces, so keep the single quotes:

```bash
terraform state mv 'github_team_membership.membership["alice/Old Name"]' 'github_team_membership.membership["alice/New Name"]'
```

The memberships need no other change: `github_team_membership.team_id` holds the team's **numeric** id, which a rename does not change.

### 4. Re-plan PR 1 and merge it

Re-trigger the plan on PR 1 (an empty commit is enough). **Expected plan comment:**

```
Terraform plan: 0 to import, 0 to add, 1 to change, 0 to destroy.
```

with a `Resources to update:` section and **no** `Resources to recreate:` section. The plan job counts recreates into both the add and the destroy totals, so `0 to destroy` is the check that matters.

> [!NOTE]
> Organisation resources have no display-name mapping in the plan summary, so they render as `unknown type github_team. resource address: github_team.team["New Name"]`. That is expected, not a failure.

Merge PR 1. The apply renames the team on GitHub and its slug becomes `new-slug`.

### 5. PR 2 — repository files only

> [!WARNING]
> Between the apply of PR 1 and the merge of PR 2, `repos/*.yaml` still points at `old-slug`, which no longer resolves. Organisation and repository config share **one workspace and one state**, so this fails the plan for *every* repository PR, not just the affected ones. **Open and approve PR 2 before merging PR 1, and merge it immediately afterwards.**

Find every reference before you start:

```bash
grep -rn 'old-slug' repos/ organisation/
```

Replace `old-slug` with `new-slug` in:

- `repos/*.yaml` — `admin_teams`, `push_teams`, `pull_teams`, `maintain_teams`, `triage_teams`
- repository ruleset `bypass_actors` entries of the form `team/old-slug`
- environment `reviewers.teams`

**Expected plan:** no changes. The team's repository grants (`github_team_repository`) are keyed by the numeric team id, so they are untouched; only the `data.github_team` lookup key changes.

> [!TIP]
> If the new name produces the **same** slug (for example `Platform` → `platform`), PR 2 is unnecessary — but steps 2–4 still are.

## 👤 Changing a member's username

This case is **reactive**: the person renamed their GitHub account, and GCSS has to follow. Every reference to the old login is already broken, so there is nothing to sequence between organisation and repository files — it is **one PR**.

`username` is `ForceNew` on `github_membership`, `github_team_membership` and `github_repository_collaborator`, so `terraform state mv` does **not** help: after the move the state still holds the old username and the plan is a replacement whose destroy removes the person from the organisation.

Use `terraform state rm` instead. The recreate is safe because the provider creates all three with an idempotent `PUT` against a user who is already a member — the role is set, nothing is revoked and no invitation is sent.

### 1. Pre-flight: protected owners

If the member is listed in the `protected_owners` input passed to `tf-plan.yaml` (a deployment-side repository variable, not config-repo content), **update that variable to the new username first**. Otherwise `validate-org` fails the PR because a protected owner is missing from `members.yaml`.

### 2. Open the PR and get it approved

One PR, updating every reference:

- `organisation/members.yaml` — `username`
- `repos/*.yaml` — `admin_collaborators`, `push_collaborators`, `pull_collaborators`, `maintain_collaborators`, `triage_collaborators`
- environment `reviewers.users`

```bash
grep -rn 'oldname' repos/ organisation/
```

Its first plan will show a destroy; that is expected until step 3. Do not merge yet.

### 3. Find every affected state address

```bash
terraform state list | grep -E '\["oldname"\]|\["oldname/'
```

That covers:

- `github_membership.member["oldname"]`
- `github_team_membership.membership["oldname/<Team Name>"]` — one per team they belong to
- `module.repository["<repo>"].github_repository_collaborator.collaborator["oldname"]` — one per repository that lists them as a collaborator

### 4. Remove them from state

One batched call, so it takes the lock once:

```bash
terraform state rm 'github_membership.member["oldname"]' 'github_team_membership.membership["oldname/Platform"]' 'module.repository["my-repo"].github_repository_collaborator.collaborator["oldname"]'
```

Nothing is deleted on GitHub — the resources are only forgotten by Terraform.

### 5. Re-plan and merge

Re-trigger the plan on the PR. **Expected plan comment:**

```
Terraform plan: 0 to import, 3 to add, 0 to change, 0 to destroy.
```

Creates only, and `0 to destroy`. The apply reconciles Terraform's state with what is already true on GitHub; the member keeps their access throughout.

## ✅ What "done" looks like

| Check | Team rename | Username change |
|---|---|---|
| Plan comment | `0 to destroy`, no `Resources to recreate:` | `0 to destroy`, creates only |
| After apply | team renamed on GitHub, membership and repo grants intact | member still active (not invited), teams and repo access intact |
| Follow-up | PR 2 merged, repo plans green again | none |
