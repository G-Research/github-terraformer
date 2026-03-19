# Adding New Features to the Importer

Quick guide for adding feature-gated functionality to the importer.

## Architecture

- Features controlled by `config/import-config.yaml`
- No CLI changes needed - purely config-driven
- Single binary works for all features

## Adding a Feature: 5 Steps

### Example: Adding `feature_github_webhooks`

#### 1. Add Constant

`pkg/github/constants.go`:

```go
const (
    FeatureGithubEnvironment = "feature_github_environment"
    FeatureGithubWebhooks    = "feature_github_webhooks"  // NEW
)
```

#### 2. Add Logic

`pkg/github/github.go` in `ImportRepo()`:

```go
// =========================================================================
// FEATURE: GitHub Webhooks
// =========================================================================
var allWebhooks []*github.Hook
if cfg != nil && cfg.IsFeatureEnabled(FeatureGithubWebhooks) {
    webhooks, _, err := v3client.Repositories.ListHooks(
        context.Background(), owner, repo, nil)
    if err != nil {
        fmt.Printf("failed to get webhooks: %v\n", err)
    } else {
        allWebhooks = webhooks
        dumpManager.WriteJSONFile("webhooks.json", webhooks)
    }
}
```

#### 3. Add Data Structure

`pkg/github/repositories.go`:

```go
type Repository struct {
    // ... existing fields ...
    Webhooks []Webhook `yaml:"webhooks,omitempty"`  // NEW
}

type Webhook struct {
    URL         string   `yaml:"url"`
    ContentType string   `yaml:"content_type,omitempty"`
    Events      []string `yaml:"events,omitempty"`
    Active      bool     `yaml:"active"`
}
```

#### 4. Add Resolver Function

`pkg/github/github.go`:

```go
func resolveWebhooks(hooks []*github.Hook) []Webhook {
    // Convert GitHub API response to YAML structure
    var webhooks []Webhook
    for _, hook := range hooks {
        webhooks = append(webhooks, Webhook{
            URL:         hook.GetURL(),
            ContentType: hook.Config["content_type"].(string),
            Events:      hook.Events,
            Active:      hook.GetActive(),
        })
    }
    return webhooks
}
```

#### 5. Document in Config

`gcss-config-repo/config/import-config.yaml`:

```yaml
# feature_github_webhooks: Import repository webhooks
#   - Webhook secrets are NOT imported (API limitation)
#   - Default: false
feature_github_webhooks: true  # Enable the feature
```

## Usage

```bash
# Enable in import-config.yaml, then:
go run main.go import owner/repo
# or
go run main.go bulk-import
```

## Key Patterns

### Always Check Config

```go
if cfg != nil && cfg.IsFeatureEnabled(FeatureYourFeature) {
    // Your code
}
```

### Error Handling

- Don't fail entire import on feature errors
- Log with `fmt.Printf`
- Continue with other features

### Data Dumps

```go
dumpManager.WriteJSONFile("feature_data.json", data)
// Saves to: dumps/<repo-name>/feature_data.json
```

## Real Example: GitHub Environments

Current implementation shows the new `deployment_policy` structure:

```go
// Data structure (pkg/github/repositories.go)
type Environment struct {
    Environment      string           `yaml:"environment"`
    WaitTimer        *int             `yaml:"wait_timer,omitempty"`
    DeploymentPolicy *DeploymentPolicy `yaml:"deployment_policy,omitempty"`
}

type DeploymentPolicy struct {
    PolicyType     string   `yaml:"policy_type"`  // "protected_branches" or "selected_branches_and_tags"
    BranchPatterns []string `yaml:"branch_patterns,omitempty"`
    TagPatterns    []string `yaml:"tag_patterns,omitempty"`
}

// Logic (pkg/github/github.go ~line 940)
if env.DeploymentBranchPolicy != nil {
    deploymentPolicy := &DeploymentPolicy{}

    if env.DeploymentBranchPolicy.ProtectedBranches != nil &&
       *env.DeploymentBranchPolicy.ProtectedBranches {
        deploymentPolicy.PolicyType = "protected_branches"
    } else if env.DeploymentBranchPolicy.CustomBranchPolicies != nil &&
              *env.DeploymentBranchPolicy.CustomBranchPolicies {
        deploymentPolicy.PolicyType = "selected_branches_and_tags"
        // Fetch patterns from API
        branchPatterns, tagPatterns := fetchDeploymentPolicies(...)
        deploymentPolicy.BranchPatterns = branchPatterns
        deploymentPolicy.TagPatterns = tagPatterns
    }

    if deploymentPolicy.PolicyType != "" {
        environment.DeploymentPolicy = deploymentPolicy
    }
}
```

## Testing

```bash
# 1. Feature disabled (default)
go run main.go import owner/repo
# Should NOT create dumps/owner-repo/webhooks.json

# 2. Feature enabled
echo "feature_github_webhooks: true" >> import-config.yaml
go run main.go import owner/repo
# Should create dumps/owner-repo/webhooks.json

# 3. Bulk import
go run main.go bulk-import
# Respects feature flag for all repos
```

## Do's and Don'ts

✅ **DO**

- Use `cfg.IsFeatureEnabled()`
- Handle errors gracefully
- Write dumps for debugging
- Document in import-config.yaml

❌ **DON'T**

- Add CLI flags or parameters
- Fail on missing data
- Skip nil checks
- Forget documentation

## Quick Reference

| File | Purpose |
|------|---------|
| `pkg/github/constants.go` | Define feature constant |
| `pkg/github/github.go` | Add feature logic in ImportRepo() |
| `pkg/github/repositories.go` | Define data structures |
| `config/import-config.yaml` | Document & enable feature |

That's it! Follow these 5 steps and your feature will integrate seamlessly.
