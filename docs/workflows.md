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

> [!IMPORTANT]
> The `schedule` environment **must not** have required-reviewer or wait-timer protection rules. The workflow runs unattended on a schedule, so any approval gate makes every run stall forever.

Caller also passes `reviewers` (comma-separated users or `org/team` slugs) to request on the drift PR.

#### Notes / limitations

- **Scale**: each run does a **full org import** plus a plan. On large orgs this is the dominant cost (several API calls per repo). Pick a cron interval that comfortably exceeds a run's duration — overlapping runs are queued (`cancel-in-progress: false`), so too-frequent scheduling lags detection.
- **Deleted / archived managed repos**: these can't be represented as a config change, so they won't appear in the drift PR. `terraform plan` still flags them and the `Inspect drift` step emits a warning, but resolving them (remove config or recreate the repo) is manual.
- GitHub disables scheduled workflows after long repo inactivity — a disabled `drift-check` means no detection.

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