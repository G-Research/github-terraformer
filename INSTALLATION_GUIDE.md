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
* Permission to create:

    * GitHub Apps
    * HCP Workspaces
    * GitHub repositories from templates
    * Deployment environments
    * Organization/team tokens in HCP

---

# 1. Create Required GitHub Apps

Two GitHub Apps must be created.

---

## 1.1 GitHub App: `<github org name> GCSS admin bypasser`

### Installation Scope

You will install this app only to one repository. See here: ([Install GitHub App](#43-install-github-app))

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

These will later be uploaded as **GitHub Deployment Environment Secrets**.

---

## 1.2 GitHub App: `<github org name> Github configuration`

### Installation Scope

Install to:

```
All repositories in the organization
```

---

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

---

# 2. Create HCP Terraform Workspace

---

## 2.1 Workspace Creation

Create a new workspace in **HCP Terraform** with a name set to the format `gcss-prod-[your org name here]-cli``:

* Workspace Type:

```
CLI-driven workspace
```

---

## 2.2 Terraform Variables

Add the following Terraform variables:

| Variable              | Type                  | Notes                                                                     |
|-----------------------| --------------------- |---------------------------------------------------------------------------|
| app_id                | Terraform             | GitHub App ID (the Github Configuration app)                              |
| app_installation_id   | Terraform             | GitHub App Installation ID                                                |
| app_private_key       | Terraform (Sensitive) | GitHub App Private Key                                                    |
| environment_directory | Terraform             | `dev` or `prod`, whatever you've set as part of the name of the workspace |
| owner                 | Terraform             | GitHub organization name                                                  |

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

# 3. Create Configuration Repository

---

## 3.1 Create Repository from Template

Template is available here:

```
https://github.com/G-Research/github-terraformer-configuration-template
```

Create repository in your organization.

Choose visibility:

* Public OR
* Private

---

# 4. Configure Repository Settings

---

## 4.1 Pull Request Settings

Path:

```
Settings → General → Pull Requests
```

Configure:

* Enable → Allow squash merging
* Disable → Other merge methods
* Enable → Automatically delete head branches

---

## 4.2 Access Review

Path:

```
Settings → Collaborators and teams
```

Review repository access.

---

## 4.3 Install GitHub App

Install:

```
<github org> GCSS admin bypasser
```

Install to:

```
Your configuration repository (this repository that you've just created) only
```

---

## 5. Configure the repository

Before creating rulesets, create a PR that:

* configures the app-list.yaml file
  * This file lists all GitHub Apps that are installed in the organization. You can either grab the list via API or manually add the apps. 
* configures the import-config.yaml
  * This config file configures the behavior of the importer workflow. Usually, you would add this repo to the ignore list.

---

## 6. Configure Branch Protection Ruleset

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
  <github org> GCSS admin bypasser
  ```

**Target Branches**

* Default branch (typically `main`)

**Branch Rules**
Enable:

* Restrict deletions
* Require pull request before merging:

    * 1 required approval
    * Dismiss stale approvals when new commits are pushed
    * Require approval of the most recent reviewable push
    * Allowed merge method → Squash only
  
* Require status checks to pass

    * Require branches to be up to date before merging
    * Add status check: `Terraform plan`, set source to: `GCSS admin bypasser` GitHub App

* Block force pushes

# 7. Configure GitHub Actions Permissions

Path:

```
Settings → Actions → General
```

Configure:

* Workflow permissions → Read and Write
* Allow GitHub Actions to:

    * Create Pull Requests
    * Approve Pull Requests

If unavailable, check Organization-level settings.

---

# 8. Create Deployment Environments

Create the following environments:

---

## 8.1 Environment: `plan`

Protection:

* Allow administrators to bypass protection rules *(temporary)*

Deployment branches:

```
main
```

Secrets:

| Name            | Value                           |
| --------------- | ------------------------------- |
| APP_PRIVATE_KEY | GCSS admin bypasser private key |
| TFC_TOKEN       | HCP Team API Token              |

Variables:

| Name      | Value                      |
| --------- | -------------------------- |
| APP_ID    | GCSS admin bypasser App ID |
| WORKSPACE | HCP workspace name         |

---

## 8.2 Environment: `schedule`

Protection:

* Allow administrators to bypass *(temporary)*

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

## 8.3 Environment: `import`

Protection:

* Allow administrators to bypass *(temporary)*

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

## 8.4 Environment: `create-fork`

Protection:

* Allow administrators to bypass *(temporary)*

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

## 8.5 Environment: `promote`

Protection:

* Allow administrators to bypass *(temporary)*

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
| APP_PRIVATE_KEY | GCSS admin bypasser private key |
| TFC_TOKEN       | HCP Team API Token              |

Variables:

| Name      | Value                      |
| --------- | -------------------------- |
| APP_ID    | GCSS admin bypasser App ID |
| WORKSPACE | HCP workspace name         |

---

# 9. Add repository level variable

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

After completing installation:

* Validate Terraform plan workflow runs
* Verify GitHub App authentication
* Confirm HCP workspace connectivity
