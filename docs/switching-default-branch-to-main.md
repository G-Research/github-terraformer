> [!IMPORTANT]
> Switching a repository's default branch from a non-`main` branch (e.g. `master`) **back to `main`** is **not a normal config change**. Editing the YAML alone produces a plan that cannot apply. Follow this runbook.

## Why

When `default_branch` is non-`main`, the module manages that branch as a real `github_branch` resource (injected into `branches_map` in `modules/terraform-github-repository/main.tf`) so it can be created declaratively. Changing `default_branch` back to `main` drops it, so Terraform plans to **destroy `github_branch.branch["master"]`** — but it destroys the branch *before* demoting the default, and GitHub refuses:

```
422 Cannot delete the default branch
```

`master` is never actually deleted, but the apply **fails and keeps failing** until state is reconciled. The fix: **`terraform state rm` the branch first** — this forgets it *without deleting it*, so `master` survives as an ordinary branch and the apply only flips the default.

## Steps

Prereq: an HCP Terraform token for the config repo's workspace (`state rm` makes no GitHub API calls — no App/PEM creds, no `-var-file`).

```bash
git clone https://github.com/G-Research/github-terraformer.git && cd github-terraformer/feature/github-repo-provisioning
ln -sf backend.tf.hcp backend.tf
export TF_CLOUD_ORGANIZATION=<TFC_ORG> TF_WORKSPACE=<WORKSPACE>   # config repo's tfc_org input and WORKSPACE variable
terraform init -input=false
terraform state pull > backup.tfstate   # always back up first
```

1. **Open a PR** changing `default_branch: master` → `main` in `repos/<repo>.yaml`, and get it approved. Its first plan shows `1 to destroy` (`github_branch.branch["master"]`) — **do not merge**.
2. **Remove the branch from state** (nothing is deleted on GitHub):
   ```bash
   terraform state rm 'module.repository["<repo>"].github_branch.branch["master"]'
   ```
3. **Re-plan and merge.** The plan is now `0 to destroy` (only `github_branch_default` updates `master` → `main`). Merge; the apply repoints the default and `master` stays put.

> [!CAUTION]
> Between the `state rm` and the merge, state no longer manages `master` but `main` still says `default_branch: master` — any other config PR merging in that window re-creates the branch and collides (`422 Reference already exists`). Approve first, then `state rm`, then merge immediately. Ask for a merge freeze if the org is busy.

**Done:** `default_branch` is `main`, `master` still exists (now unmanaged — re-`import` it if you want it managed again). Roll back via the HCP state history or `terraform import 'module.repository["<repo>"].github_branch.branch["master"]' '<repo>:master'`.
