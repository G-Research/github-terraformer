## Table of Contents

- [📦 Github terraformer](#-github-terraformer)
    - [🆕 Creating New Repositories](#-creating-new-repositories)
    - [🍴 Handling Forks](#-handling-forks)
    - [📚 Documentation](#-documentation)

# 📦 Github terraformer

This repository automates the management of GitHub repositories within your GitHub organization using Terraform Cloud and GitHub Actions. It supports both the **import of existing repositories** and the **provisioning of new repositories**, ensuring GitHub configuration is reproducible, auditable, and version-controlled. This repository contains reusable workflows that act as an API that you should use in your repository (aka config repo) that should contain configuration files of repositories of your organization. Config repo is a repository that contains yaml configuration files for each repository in your organization, and should be excluded from Terraform management.

## 🆕 Creating New Repositories

To provision a **brand-new repository** in your GitHub organization:

1. Clone your config repo.
2. Create a new branch.
3. Add a YAML file describing the desired repository configuration - name of the YAML file will be the name of the repository (case-sensitive)
3. Submit the PR for review
4. Upon approval and merge, Terraform Cloud will:
    - Plan and apply the configuration
    - Create the repository

> 📝 New repositories must not be forks. Forks follow a different workflow (see below).

## 🍴 Handling Forks

To import a **forked** repository into the organization:

1. Trigger the Create fork workflow on your config repo
2. Provide input:
    - Repo to fork (in the format of `owner/repo`)
    - Name of the new forked repository. If left empty, default would be the same as the upstream repo name
    - To create the fork with only the default branch (e.g., `main` or `master`) or all branches. Default is `true`
3. Workflow will fork and import the repository triggering the import workflow. From here:
    1. The workflow will generate a YAML config for the forked repository
    2. Place it under `feature/github-repo-provisioning/importer_tmp_dir/`
    3. Create a PR against the `main` branch
    4. _User reviews, approves, and merges the PR_
4. After merge Terraform will import the forked repository into its state by applying the generated YAML configuration
5. The configuration file is then sanitized (ids removed) and moved to the appropriate directory
6. From here, user can make changes to the configuration file as needed, and should create a PR against the `{repository}.yaml` file to apply further changes

> 📝 We are working on improving this so that the user has the same experience as when creating a new repo

## 📚 Documentation

- [Developer's Guide](DEVELOPERS_GUIDE.md) — every attribute available in a `repos/<repo>.yaml` config
- [Installation Guide](INSTALLATION_GUIDE.md) — one-time setup of the GitHub Apps, HCP workspace and config repo
- [Workflows](docs/workflows.md) — the reusable import and drift-check workflows
- [Renaming teams and members](docs/renaming-teams-and-members.md) — how to rename without destroying and recreating

> [!IMPORTANT]
> All important attributes are documented in the [Developer's Guide](DEVELOPERS_GUIDE.md).