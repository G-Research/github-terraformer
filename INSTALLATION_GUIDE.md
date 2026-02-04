# GitHub Terraformer – Installation Guide

## Overview

GitHub Terraformer is a tool used to manage GitHub repositories through Terraform, executed via **HCP Terraform (HashiCorp Cloud Platform)** and orchestrated through GitHub workflows.

This guide covers:

* GitHub App setup
* HCP Terraform workspace configuration
* Configuration repository setup
* Repository rulesets and environments

---

## Prerequisites

You must have:

* Your GitHub Organization admin access
* HCP Terraform organization access

---

# 1. Create `<github org name> Github configuration` Github app

### Repository Permissions

| Permission        | Access              |
| ----------------- | ------------------- |
| Actions           | Read & Write        |
| Administration    | Read & Write        |
| Checks            | Read & Write        |
| Contents          | Read & Write        |
| Dependabot Alerts | Read & Write        |
| Metadata          | Read-only |
| Pages             | Read & Write        |
| Pull Requests     | Read & Write        |

---

### Organization Permissions

| Permission     | Access    |
| -------------- | --------- |
| Administration | Read-only |
| Members        | Read-only |

---

### Credentials Handling

After creating the app:

1. Generate a **Private Key**
2. Save locally:

    * App ID
    * Private Key (.pem)
    * Installation ID (this is available after installing the app to the organization)

These will later be uploaded as **GitHub Deployment Environment Secrets**.

### Installation Scope

Install to:
```
All repositories in the organization
```

---

# 2. Create HCP Terraform Workspace

## 2.1 Workspace Creation

Create a new workspace in **HCP Terraform** with a name set to the format `github-configuration-prod-[your org name here]-cli``:

* Workspace Type:

```
CLI-driven workspace
```

---

## 2.2 Terraform Variables

Add the following Terraform variables:

| Variable              | Type                  | Notes                                                                                              |
|-----------------------| --------------------- |----------------------------------------------------------------------------------------------------|
| app_id                | Terraform             | GitHub App ID (the Github Configuration app)                                                       |
| app_installation_id   | Terraform             | GitHub App Installation ID                                                                         |
| app_private_key       | Terraform (Sensitive) | GitHub App Private Key                                                                             |
| environment_directory | Terraform             | `dev` or `prod`, whatever you've set as part of the name of the workspace. This will be deprecated |
| owner                 | Terraform             | GitHub organization name                                                                           |

---

## 2.3 Workspace Settings

In **General Settings**:

```
User Interface → Console UI
```

---

## 2.4 Create HCP Team API Token

In the HCP organization that owns the workspace:

1. Go to **Team Tokens**
2. Create new API Token
3. Store locally

This token will later be added as a GitHub Deployment Environment secret.

---

# 3. Create Configuration Repository from Template

Template is available here:

```
https://github.com/G-Research/github-terraformer-configuration-template
```

Create repository in your organization.

Choose visibility:

* Public OR
* Private

---

# 4. Create `GitHub Terraformer workflow bot` GitHub App

### Repository Permissions

| Permission    | Access       |
| ------------- | ------------ |
| Checks        | Read & Write |
| Contents      | Read & Write |
| Metadata      | Read-only    |
| Pull Requests | Read & Write |

---

### Credentials Handling

After creating the app:

1. Generate a **Private Key**
2. Save locally:

    * App ID
    * Private Key (.pem)

These will be uploaded as **GitHub Deployment Environment Secrets**.

---

### Installation Scope

You will install this app only to the configuration repository.

---

# 5. Configure Repository Settings

## 5.1 Pull Request Settings

Path:

```
Settings → General → Pull Requests
```

Configure:

* Enable → Allow squash merging
* Disable → Other merge methods
* Enable → Automatically delete head branches

---

## 5.2 Access Review

Path:

```
Settings → Collaborators and teams
```

Give write access to teams and/or collaborators that are expected to use this repository to make configuration changes

---

## 6. Configure the repository

Before creating rulesets, create a PR that:

* configures the app-list.yaml file
  * This file lists all GitHub Apps that are installed in the organization. You can either grab the list via API or manually add the apps, or execute this command: 
  ```
    gh api orgs/<ORG>/installations --paginate \ 
        --jq '{apps: [.installations[] | {app_owner: .account.login, app_id: .app_id, app_slug: .app_slug}]}' \
        | yq -P
  ```
* configures the import-config.yaml
  * This config file configures the behavior of the importer workflow. Usually, you would add this repo to the ignore list.

---

## 7. Configure Branch Protection Ruleset

Create a new ruleset with the following configuration.

**Ruleset Name**

```
Protect main branch
```

**Enforcement**

* Status → Active

**Bypass List**

* Add GitHub App:

  ```
  GitHub Terraformer workflow bot
  ```

**Target Branches**

* Default branch (typically `main`)

**Branch Rules**
Enable:

* Restrict deletions
* Require pull request before merging (the following settings are only suggested, you can adjust them to your needs):

    * 1 required approval
    * Dismiss stale approvals when new commits are pushed
    * Require approval of the most recent reviewable push
    * Allowed merge method → Squash only
  
* Require status checks to pass

    * Require branches to be up to date before merging
    * Add status check: `Terraform plan`, set source to: `GitHub Terraformer workflow bot` GitHub App

* Block force pushes

---

# 8. Configure GitHub Actions Permissions

Path:

```
Settings → Actions → General
```

Configure:

* Workflow permissions → Read and Write

If unavailable, check Organization-level settings.

---

# 9. Create Deployment Environments

Create the following environments:

## 9.1 Environment: `plan`

Deployment branches:

```
main
```

Secrets:

| Name            | Value                           |
| --------------- | ------------------------------- |
| APP_PRIVATE_KEY | GitHub Terraformer workflow bot private key |
| TFC_TOKEN       | HCP Team API Token              |

Variables:

| Name      | Value                      |
| --------- | -------------------------- |
| APP_ID    | GitHub Terraformer workflow bot App ID |
| WORKSPACE | HCP workspace name         |

---

## 9.2 Environment: `schedule`

Deployment branches:

```
main
```

Secrets:

| Name      | Value              |
| --------- | ------------------ |
| TFC_TOKEN | HCP Team API Token |

Variables:

| Name      | Value              |
| --------- | ------------------ |
| WORKSPACE | HCP workspace name |

---

## 9.3 Environment: `import`

Deployment branches:

```
main
```

Secrets:

| Name            | Value                                    |
| --------------- | ---------------------------------------- |
| APP_PRIVATE_KEY | `<org> Github configuration` private key |

Variables:

| Name   | Value                               |
| ------ | ----------------------------------- |
| APP_ID | `<org> Github configuration` App ID |

---

## 9.4 Environment: `create-fork`

Deployment branches:

```
main
```

Secrets:

| Name            | Value                                    |
| --------------- | ---------------------------------------- |
| APP_PRIVATE_KEY | `<org> Github configuration` private key |

Variables:

| Name   | Value                               |
| ------ | ----------------------------------- |
| APP_ID | `<org> Github configuration` App ID |

---

## 9.5 Environment: `promote`

Deployment branches:

```
main
import/*/*/*
import/main/<org-name>/bulk-import/*
refs/pull/*/merge
```

Secrets:

| Name            | Value                           |
| --------------- | ------------------------------- |
| APP_PRIVATE_KEY | GitHub Terraformer workflow bot private key |
| TFC_TOKEN       | HCP Team API Token              |

Variables:

| Name      | Value                      |
| --------- | -------------------------- |
| APP_ID    | GitHub Terraformer workflow bot App ID |
| WORKSPACE | HCP workspace name         |

---

# 10. Add repository level variable

Path:

```
Security → Secrets and variables → Actions → Variables tab 
```

Add new repository variable:

| Name      | Value                  |
|-----------|------------------------|
| TFC_ORG   | Your org name from HCP |

---

# Next Steps

After completing installation, you should be able to run the `import`/`bulk-import` workflow which will confirm if:

* GitHub App authentication is set up properly
* Confirm HCP workspace connectivity
