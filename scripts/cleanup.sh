#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m'

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  GitHub Terraformer - Installation Cleanup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

if [ ! -d .install-data ]; then
    log_warning "No installation data found (.install-data/ directory doesn't exist)"
    echo
    exit 0
fi

log_warning "This will remove all saved installation data including:"
echo "  - App IDs and installation IDs"
echo "  - Private key files"
echo "  - HCP Terraform tokens"
echo "  - Workspace names"
echo

log_warning "This does NOT remove:"
echo "  - Created GitHub Apps (delete manually from GitHub)"
echo "  - Created repositories (delete manually from GitHub)"
echo "  - HCP Terraform workspaces (delete manually from HCP)"
echo "  - Deployed secrets and variables (will be overwritten on next install)"
echo

read -p "$(echo -e "${YELLOW}?${NC} Are you sure you want to continue? [y/N]: ")" response

if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled"
    exit 0
fi

echo
log_warning "Removing installation data..."

rm -rf .install-data

log_success "Installation data removed"
echo
echo "You can now run ./scripts/install.sh to start a fresh installation"
echo
