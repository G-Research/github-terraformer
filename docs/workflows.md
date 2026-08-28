> [!IMPORTANT]
> This is a work in progress document and may change in the future

## 🚀 GitHub Actions Workflows

### 🔄 `Import` Workflow

- **Trigger**: Manually via GitHub Actions
- **Inputs**:
    - `branch`: Target environment (`dev` or `prod`)
    - `repo_name`: Name of the GitHub repository to import
    - `owner`: Name of the Github organization that owns the repository
- **Behavior**:
    1. Fetches repo metadata via GitHub API:
        - General repository settings
        - Branch protection rules
        - Default branch
        - Teams and collaborators
        - Repository rulesets
    2. Generates a YAML configuration
    3. Places the YAML into:
       ```
       feature/github-repo-provisioning/importer_tmp_dir/{organization}/{repository}.yaml
       ```
    4. Creates an automated pull request targeting the selected branch
    5. Upon PR merge, Terraform Cloud plans and applies the configuration
    6. Configuration file is then sanitized (ids removed) and moved to the appropriate directory `feature/github-repo-provisioning/repo_configs/{branch}/{organization}`

### ✅ `Validate and Plan` Workflow

- **Trigger**: `workflow_run` from the config repo, on every pull request.
- **Behavior**: a single `validate-and-plan` job, top to bottom:
    1. Opens a check run named **`Terraform plan`** on the head commit, `in_progress`, before any work happens.
    2. Validates `repos/*.yaml` against the repository schema and `organisation/*.yaml` against the teams/members schemas and cross-file rules — including the protected-owner rule, which fails the PR if a protected identity is removed from or demoted in `members.yaml`.
    3. Runs `terraform plan` on Terraform Cloud. Like every later step it is skipped once validation has failed, so an invalid config never reaches Terraform.
    4. `Report result` runs on every path and concludes that same check run exactly once: `success` with the plan summary, `failure` for a validation or plan failure, `cancelled` for a cancelled run. It posts the plan summary as a PR comment whenever the plan actually ran.

`Terraform plan` is the check to list as a required status check. It is opened up front and always concluded, so a failed validation now reports a failed check instead of leaving the pull request waiting on a check that never arrives.

#### Consumer setup requirements

The job runs in the **`plan`** environment. Variables are read from that environment by the workflow itself:

| Name | Purpose |
|---|---|
| `PROTECTED_OWNERS` | Comma-separated org logins that must stay owners in `organisation/members.yaml` |
| `APP_ID` | GitHub App that posts the check run and the plan comment (needs **Checks: write**) |
| `WORKSPACE` | Terraform Cloud workspace |

The caller must also pass all three `workflow_call` secrets — these are declared on the reusable workflow, not looked up by the job:

| Name | Purpose |
|---|---|
| `app_private_key` | Private key for the App named by `APP_ID` |
| `gh_token` | Checks the config repo out at the commit under review |
| `tfc_token` | Terraform Cloud API token |

> [!IMPORTANT]
> `PROTECTED_OWNERS` is deployment config and is deliberately read from the environment rather than passed in by the caller, so that a pull request cannot weaken the rule it is validated against. Leaving it unset is not a way to opt out: whenever any `organisation/` config is present, `validate-org` fails. A config repo with no organisation config at all needs no list and passes.

> [!WARNING]
> The `plan` environment **must not** have required-reviewer or wait-timer protection rules. Everything, validation included, runs in this environment — so an approval gate holds up the whole pull request, and the check run does not even reach `in_progress` until someone approves.

### 🔍 `Drift Check` Workflow

- **Trigger**: Scheduled (cron) from the config repo.
- **Behavior**:
    1. Imports the current GitHub state of every repo in the org.
    2. `compare` drops everything that matches the committed config, leaving only **changes** — repos created outside GCSS *and* manual edits to already-managed repos.
    3. Runs `terraform plan` (with `-refresh`) over the result.
    4. Opens / updates / closes a single PR (`drift/detected-changes`) with the detected changes, assigned to the configured reviewers. Reviewers either **merge** (adopt the change into config) or **revert** the change manually.

#### Consumer setup requirements

The reusable `drift-check.yaml` runs in the **`schedule`** environment. That environment **must** provide all of:

| Name | Type | Purpose |
|---|---|---|
| `APP_ID` | variable | Management GitHub App (must have **org-wide "All repositories" access**, or new repos are invisible to the importer) |
| `app_private_key` | secret | Private key for that App |
| `WORKSPACE` | variable | Terraform Cloud workspace |
| `tfc_token` | secret | Terraform Cloud API token |
| `DRIFT_REVIEWERS` | variable | Comma-separated users or `org/team` slugs to request as reviewers on the drift PR. When unset the drift PR still opens, unassigned, and the run logs a warning |

> [!IMPORTANT]
> The `schedule` environment **must not** have required-reviewer or wait-timer protection rules. The workflow runs unattended on a schedule, so any approval gate makes every run stall forever.

#### Notes / limitations

- **Scale**: each run does a **full org import** plus a plan. On large orgs this is the dominant cost (several API calls per repo). Pick a cron interval that comfortably exceeds a run's duration — overlapping runs are queued (`cancel-in-progress: false`), so too-frequent scheduling lags detection.
- **Deleted / archived managed repos**: these can't be represented as a config change, so they won't appear in the drift PR. `terraform plan` still flags them and the `Inspect drift` step emits a warning, but resolving them (remove config or recreate the repo) is manual.
- GitHub disables scheduled workflows after long repo inactivity — a disabled `drift-check` means no detection.

### 🗑 `Decommission` Workflow

Forgets a repository that has **left the organization** — transferred out or deleted — from Terraform state **without destroying it**. Deleting its config alone would make Terraform plan a destroy of a repository the organization no longer owns, and because state is shared, one orphaned repository blocks plans for every other one.

- **Trigger**: Manual (`workflow_dispatch`) from the config repo, with the repository name.
- **Behavior**:
    1. `discover` lists every state address the repository owns — `module.repository["<repo>"]` plus root-level resources matched by their `repository` attribute, so hashed ruleset keys and composite-keyed environment resources are never missed.
    2. `decommission` waits for approval, then runs `terraform state rm` on that list.
    3. A final step re-reads state and fails if anything still references the repository.

> [!IMPORTANT]
> Run this **while `repos/<repo>.yaml` is still present**, then delete the config afterwards through a normal pull request. `discover` refuses to run without it. Because the resources are already out of state by then, that later apply is a no-op.

#### Consumer setup requirements

| Name | Type | Scope | Purpose |
|---|---|---|---|
| `WORKSPACE` | variable | **repository** | Terraform Cloud workspace |
| `TFC_TOKEN` | secret | **repository** | Terraform Cloud API token, needing write access to the workspace — `state rm` modifies state |
| `tfc_token` | secret | `decommission` environment | The same token, for the job that runs behind the approval gate |

> [!WARNING]
> Both must be **repository-level**, unlike every other workflow here. The `discover` job and the concurrency group run outside any deployment environment and cannot read environment-scoped values. A token available only to the `decommission` environment leaves `discover` unable to reach the workspace at all.

The `decommission` environment **must** have required reviewers. That approval is the only gate before state is modified.

#### Notes / limitations

- **State only.** The repository on GitHub is never touched — this removes Terraform's record of it, nothing more.
- **Terraform version.** Both jobs read the version the workspace requires and install that. `plan` and `apply` are unaffected by version drift because they execute remotely, but `state rm` runs locally against remote state and is refused if the CLI does not satisfy the workspace constraint.
- **Concurrency** is per workspace with `cancel-in-progress: false`, so runs queue rather than interrupt a state operation.
- The address list is computed in `discover`, before the approval wait. If state changes while the run waits, the final verification step fails and the workflow must be re-run so the list is rediscovered.

## 📥 Importing Existing Repositories

To import an **existing GitHub repository** into Terraform:

1. Navigate to **Actions** > **Import** workflow in GitHub
2. Select:
    - `prod` (or `dev`) as the target branch
    - The name of the repository to import
    - The owner of the repository (e.g., `G-Research` or `armadaproject`)
3. The workflow will:
    - Generate a YAML config
    - Place it under `feature/github-repo-provisioning/importer_tmp_dir/{organization}/`
    - The name of the YAML file will be the same as the repository name
    - Create a PR against the `prod` branch
4. Review, approve, and merge the PR
5. Terraform Cloud will detect and apply the changes
