#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ ${NC}$1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

prompt_input() {
    local prompt="$1"
    local var_name="$2"
    local default="${3:-}"

    if [ -n "$default" ]; then
        read -p "$(echo -e "${BLUE}?${NC} $prompt [$default]: ")" input
        eval "$var_name=\"${input:-$default}\""
    else
        read -p "$(echo -e "${BLUE}?${NC} $prompt: ")" input
        eval "$var_name=\"$input\""
    fi
}

prompt_secret() {
    local prompt="$1"
    local var_name="$2"

    read -s -p "$(echo -e "${BLUE}?${NC} $prompt: ")" input
    echo
    eval "$var_name=\"$input\""
}

confirm() {
    local prompt="$1"
    read -p "$(echo -e "${YELLOW}?${NC} $prompt [y/N]: ")" response
    [[ "$response" =~ ^[Yy]$ ]]
}

wait_for_enter() {
    read -p "$(echo -e "${YELLOW}⏎${NC} Press Enter to continue...")"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLI (gh) is not installed. Please install it first:"
        log_error "  brew install gh"
        exit 1
    fi

    if ! gh auth status &> /dev/null; then
        log_error "GitHub CLI is not authenticated. Please run: gh auth login"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Step 1: Create GitHub Configuration App
create_github_config_app() {
    local org_name="$1"
    local app_name="${org_name} Github configuration"

    log_info "Creating GitHub App: $app_name"

    # Create a temporary manifest file
    local manifest_file=$(mktemp)
    cat > "$manifest_file" <<EOF
{
  "name": "$app_name",
  "url": "https://github.com/${org_name}",
  "hook_attributes": {
    "active": false
  },
  "public": false,
  "default_permissions": {
    "actions": "write",
    "administration": "write",
    "checks": "write",
    "contents": "write",
    "dependabot_secrets": "write",
    "metadata": "read",
    "pages": "write",
    "pull_requests": "write",
    "organization_administration": "read",
    "members": "read"
  },
  "default_events": []
}
EOF

    log_warning "You need to create this GitHub App manually through the GitHub web interface:"
    log_info "1. Go to: https://github.com/organizations/${org_name}/settings/apps/new"
    log_info "2. Fill in the following details:"
    echo
    echo "   App Name: $app_name"
    echo "   Homepage URL: https://github.com/${org_name}"
    echo "   Webhook: Uncheck 'Active'"
    echo
    echo "   Repository Permissions:"
    echo "     - Actions: Read & Write"
    echo "     - Administration: Read & Write"
    echo "     - Checks: Read & Write"
    echo "     - Contents: Read & Write"
    echo "     - Dependabot Alerts: Read & Write"
    echo "     - Metadata: Read-only"
    echo "     - Pages: Read & Write"
    echo "     - Pull Requests: Read & Write"
    echo
    echo "   Organization Permissions:"
    echo "     - Administration: Read-only"
    echo "     - Members: Read-only"
    echo
    log_info "3. After creating the app:"
    log_info "   - Generate a Private Key and download it"
    log_info "   - Note the App ID"
    log_info "   - Install the app to ALL repositories in the organization"
    log_info "   - Note the Installation ID (visible in the URL after installation)"
    echo

    rm -f "$manifest_file"
    wait_for_enter

    local app_id installation_id private_key_path
    prompt_input "Enter the GitHub Configuration App ID" app_id
    prompt_input "Enter the Installation ID" installation_id
    prompt_input "Enter the path to the downloaded private key (.pem file)" private_key_path

    if [ ! -f "$private_key_path" ]; then
        log_error "Private key file not found: $private_key_path"
        exit 1
    fi

    echo "$app_id" > .install-data/github_config_app_id
    echo "$installation_id" > .install-data/github_config_installation_id
    cp "$private_key_path" .install-data/github_config_private_key.pem

    log_success "GitHub Configuration App credentials saved"
}

# Step 2: HCP Terraform Workspace setup (manual instructions)
setup_hcp_terraform() {
    local org_name="$1"

    log_warning "HCP Terraform Workspace Setup (Manual Step)"
    echo
    log_info "Please complete the following in HCP Terraform:"
    echo
    echo "1. Create a new CLI-driven workspace named: github-configuration-prod-${org_name}-cli"
    echo
    echo "2. Add the following Terraform variables:"
    echo "   - app_id: $(cat .install-data/github_config_app_id)"
    echo "   - app_installation_id: $(cat .install-data/github_config_installation_id)"
    echo "   - app_private_key: (Sensitive) <paste the contents of .install-data/github_config_private_key.pem>"
    echo "   - environment_directory: prod"
    echo "   - owner: ${org_name}"
    echo
    echo "3. In General Settings, set User Interface to: Console UI"
    echo
    echo "4. Create a Team API Token and save it"
    echo
    wait_for_enter

    local workspace_name tfc_token tfc_org
    prompt_input "Enter the HCP Terraform workspace name" workspace_name "github-configuration-prod-${org_name}-cli"
    prompt_input "Enter your HCP Terraform organization name" tfc_org
    prompt_secret "Enter the HCP Team API Token" tfc_token

    echo "$workspace_name" > .install-data/hcp_workspace_name
    echo "$tfc_org" > .install-data/hcp_org_name
    echo "$tfc_token" > .install-data/hcp_team_token

    log_success "HCP Terraform details saved"
}

# Step 3: Create configuration repository from template
create_config_repo() {
    local org_name="$1"
    local repo_name="${org_name}-github-terraformer-config"

    log_info "Creating configuration repository from template..."

    local visibility
    prompt_input "Repository visibility (public/private)" visibility "private"

    if gh repo view "${org_name}/${repo_name}" &> /dev/null; then
        log_warning "Repository ${org_name}/${repo_name} already exists"
        if ! confirm "Use existing repository?"; then
            prompt_input "Enter a different repository name" repo_name
        fi
    else
        log_info "Creating repository ${org_name}/${repo_name} from template..."

        gh repo create "${org_name}/${repo_name}" \
            --template "G-Research/github-terraformer-configuration-template" \
            --${visibility} \
            --clone

        log_success "Repository created and cloned"
    fi

    echo "$repo_name" > .install-data/config_repo_name
}

# Step 4: Create Workflow Bot GitHub App
create_workflow_bot_app() {
    local org_name="$1"
    local app_name="GitHub Terraformer workflow bot"

    log_warning "Creating GitHub App: $app_name"
    echo
    log_info "You need to create this GitHub App manually through the GitHub web interface:"
    log_info "1. Go to: https://github.com/organizations/${org_name}/settings/apps/new"
    log_info "2. Fill in the following details:"
    echo
    echo "   App Name: $app_name"
    echo "   Homepage URL: https://github.com/${org_name}"
    echo "   Webhook: Uncheck 'Active'"
    echo
    echo "   Repository Permissions:"
    echo "     - Checks: Read & Write"
    echo "     - Contents: Read & Write"
    echo "     - Metadata: Read-only"
    echo "     - Pull Requests: Read & Write"
    echo
    log_info "3. After creating the app:"
    log_info "   - Generate a Private Key and download it"
    log_info "   - Note the App ID"
    log_info "   - Install the app ONLY to the configuration repository"
    echo
    wait_for_enter

    local app_id private_key_path
    prompt_input "Enter the Workflow Bot App ID" app_id
    prompt_input "Enter the path to the downloaded private key (.pem file)" private_key_path

    if [ ! -f "$private_key_path" ]; then
        log_error "Private key file not found: $private_key_path"
        exit 1
    fi

    echo "$app_id" > .install-data/workflow_bot_app_id
    cp "$private_key_path" .install-data/workflow_bot_private_key.pem

    log_success "Workflow Bot App credentials saved"
}

# Step 5: Configure repository settings
configure_repo_settings() {
    local org_name="$1"
    local repo_name="$2"

    log_info "Configuring repository settings for ${org_name}/${repo_name}..."

    # Enable squash merge, disable other merge methods
    gh api -X PATCH "/repos/${org_name}/${repo_name}" \
        -f allow_squash_merge=true \
        -f allow_merge_commit=false \
        -f allow_rebase_merge=false \
        -f delete_branch_on_merge=true \
        > /dev/null

    log_success "Pull request settings configured"
}

# Step 6: Generate app-list.yaml
generate_app_list() {
    local org_name="$1"
    local repo_name="$2"

    log_info "Generating app-list.yaml..."

    local repo_dir="${repo_name}"
    if [ ! -d "$repo_dir" ]; then
        log_warning "Repository directory not found. Skipping app-list.yaml generation."
        log_info "You can generate it manually later with:"
        echo "  gh api orgs/${org_name}/installations --paginate \\"
        echo "    --jq '{apps: [.installations[] | {app_owner: .account.login, app_id: .app_id, app_slug: .app_slug}]}' \\"
        echo "    | yq -P > app-list.yaml"
        return
    fi

    cd "$repo_dir"

    gh api "orgs/${org_name}/installations" --paginate \
        --jq '{apps: [.installations[] | {app_owner: .account.login, app_id: .app_id, app_slug: .app_slug}]}' \
        > app-list.json

    if command -v yq &> /dev/null; then
        yq -P < app-list.json > app-list.yaml
        rm app-list.json
    else
        log_warning "yq not installed. app-list.json created, but not converted to YAML"
        log_info "Install yq with: brew install yq"
    fi

    cd ..
    log_success "app-list configuration generated"
}

# Step 7: Create branch protection ruleset
create_branch_protection() {
    local org_name="$1"
    local repo_name="$2"
    local workflow_bot_app_id="$3"

    log_info "Creating branch protection ruleset..."

    # Note: Branch protection rulesets via API are in beta and complex
    # Providing manual instructions instead
    log_warning "Branch protection ruleset needs to be configured manually:"
    echo
    echo "Go to: https://github.com/${org_name}/${repo_name}/settings/rules/new"
    echo
    echo "Ruleset Configuration:"
    echo "  Name: Protect main branch"
    echo "  Status: Active"
    echo "  Bypass list: GitHub Terraformer workflow bot (App ID: ${workflow_bot_app_id})"
    echo "  Target branches: Default branch (main)"
    echo
    echo "  Branch Rules:"
    echo "    ☑ Restrict deletions"
    echo "    ☑ Require pull request before merging"
    echo "      - Required approvals: 1"
    echo "      - Dismiss stale approvals when new commits are pushed"
    echo "      - Require approval of the most recent reviewable push"
    echo "      - Allowed merge method: Squash only"
    echo "    ☑ Require status checks to pass"
    echo "      - Require branches to be up to date before merging"
    echo "      - Status check: 'Terraform plan' (source: GitHub Terraformer workflow bot)"
    echo "    ☑ Block force pushes"
    echo
    wait_for_enter
}

# Step 8: Configure GitHub Actions permissions
configure_actions_permissions() {
    local org_name="$1"
    local repo_name="$2"

    log_info "Configuring GitHub Actions permissions..."

    gh api -X PUT "/repos/${org_name}/${repo_name}/actions/permissions" \
        -f default_workflow_permissions=write \
        > /dev/null

    log_success "GitHub Actions permissions configured"
}

# Step 9: Create deployment environments
create_deployment_environments() {
    local org_name="$1"
    local repo_name="$2"

    log_info "Creating deployment environments..."

    local workflow_bot_app_id=$(cat .install-data/workflow_bot_app_id)
    local workflow_bot_private_key=$(cat .install-data/workflow_bot_private_key.pem)
    local github_config_app_id=$(cat .install-data/github_config_app_id)
    local github_config_private_key=$(cat .install-data/github_config_private_key.pem)
    local hcp_workspace=$(cat .install-data/hcp_workspace_name)
    local tfc_token=$(cat .install-data/hcp_team_token)

    # Environment: plan
    log_info "Creating environment: plan"
    gh api -X PUT "/repos/${org_name}/${repo_name}/environments/plan" > /dev/null

    gh secret set APP_PRIVATE_KEY --env plan --repo "${org_name}/${repo_name}" --body "$workflow_bot_private_key"
    gh secret set TFC_TOKEN --env plan --repo "${org_name}/${repo_name}" --body "$tfc_token"
    gh variable set APP_ID --env plan --repo "${org_name}/${repo_name}" --body "$workflow_bot_app_id"
    gh variable set WORKSPACE --env plan --repo "${org_name}/${repo_name}" --body "$hcp_workspace"

    log_success "Environment 'plan' created"

    # Environment: schedule
    log_info "Creating environment: schedule"
    gh api -X PUT "/repos/${org_name}/${repo_name}/environments/schedule" > /dev/null

    gh secret set TFC_TOKEN --env schedule --repo "${org_name}/${repo_name}" --body "$tfc_token"
    gh variable set WORKSPACE --env schedule --repo "${org_name}/${repo_name}" --body "$hcp_workspace"

    log_success "Environment 'schedule' created"

    # Environment: import
    log_info "Creating environment: import"
    gh api -X PUT "/repos/${org_name}/${repo_name}/environments/import" > /dev/null

    gh secret set APP_PRIVATE_KEY --env import --repo "${org_name}/${repo_name}" --body "$github_config_private_key"
    gh variable set APP_ID --env import --repo "${org_name}/${repo_name}" --body "$github_config_app_id"

    log_success "Environment 'import' created"

    # Environment: create-fork
    log_info "Creating environment: create-fork"
    gh api -X PUT "/repos/${org_name}/${repo_name}/environments/create-fork" > /dev/null

    gh secret set APP_PRIVATE_KEY --env create-fork --repo "${org_name}/${repo_name}" --body "$github_config_private_key"
    gh variable set APP_ID --env create-fork --repo "${org_name}/${repo_name}" --body "$github_config_app_id"

    log_success "Environment 'create-fork' created"

    # Environment: promote
    log_info "Creating environment: promote"
    gh api -X PUT "/repos/${org_name}/${repo_name}/environments/promote" > /dev/null

    gh secret set APP_PRIVATE_KEY --env promote --repo "${org_name}/${repo_name}" --body "$workflow_bot_private_key"
    gh secret set TFC_TOKEN --env promote --repo "${org_name}/${repo_name}" --body "$tfc_token"
    gh variable set APP_ID --env promote --repo "${org_name}/${repo_name}" --body "$workflow_bot_app_id"
    gh variable set WORKSPACE --env promote --repo "${org_name}/${repo_name}" --body "$hcp_workspace"

    log_success "Environment 'promote' created"

    log_success "All deployment environments created"
}

# Step 10: Add repository level variable
add_repo_variable() {
    local org_name="$1"
    local repo_name="$2"

    log_info "Adding repository level variable TFC_ORG..."

    local tfc_org=$(cat .install-data/hcp_org_name)
    gh variable set TFC_ORG --repo "${org_name}/${repo_name}" --body "$tfc_org"

    log_success "Repository variable TFC_ORG set to: $tfc_org"
}

# Main installation flow
main() {
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  GitHub Terraformer - Automated Installation"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo

    check_prerequisites

    # Create data directory for storing installation state
    mkdir -p .install-data
    chmod 700 .install-data

    # Get organization name
    local org_name
    prompt_input "Enter your GitHub organization name" org_name

    echo
    log_info "Starting installation for organization: $org_name"
    echo

    # Step 1: Create GitHub Configuration App
    if [ ! -f .install-data/github_config_app_id ]; then
        create_github_config_app "$org_name"
    else
        log_success "GitHub Configuration App already configured (skipping)"
    fi

    # Step 2: HCP Terraform setup
    if [ ! -f .install-data/hcp_workspace_name ]; then
        setup_hcp_terraform "$org_name"
    else
        log_success "HCP Terraform already configured (skipping)"
    fi

    # Step 3: Create configuration repository
    if [ ! -f .install-data/config_repo_name ]; then
        create_config_repo "$org_name"
    else
        log_success "Configuration repository already created (skipping)"
    fi

    local repo_name=$(cat .install-data/config_repo_name)

    # Step 4: Create Workflow Bot App
    if [ ! -f .install-data/workflow_bot_app_id ]; then
        create_workflow_bot_app "$org_name"
    else
        log_success "Workflow Bot App already configured (skipping)"
    fi

    # Step 5: Configure repository settings
    configure_repo_settings "$org_name" "$repo_name"

    # Step 6: Generate app-list.yaml
    generate_app_list "$org_name" "$repo_name"

    # Step 7: Branch protection (manual)
    local workflow_bot_app_id=$(cat .install-data/workflow_bot_app_id)
    create_branch_protection "$org_name" "$repo_name" "$workflow_bot_app_id"

    # Step 8: Configure Actions permissions
    configure_actions_permissions "$org_name" "$repo_name"

    # Step 9: Create deployment environments
    create_deployment_environments "$org_name" "$repo_name"

    # Step 10: Add repository variable
    add_repo_variable "$org_name" "$repo_name"

    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_success "Installation completed!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    log_info "Next steps:"
    echo "  1. Update app-list.yaml and import-config.yaml in the configuration repository"
    echo "  2. Create a PR with these changes"
    echo "  3. Complete the branch protection ruleset setup (if not done)"
    echo "  4. Test by running the import/bulk-import workflow"
    echo
    log_warning "Installation data saved in .install-data/ directory"
    log_warning "Keep this directory secure and do not commit it to git!"
    echo
}

main "$@"
