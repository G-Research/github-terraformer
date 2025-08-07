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