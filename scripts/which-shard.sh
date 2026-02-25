#!/bin/bash
# Determine which shard a repository belongs to

REPO_NAME="$1"
NUM_SHARDS=10

if [ -z "$REPO_NAME" ]; then
  echo "Usage: $0 <repo-name>"
  echo ""
  echo "Example: $0 my-awesome-repo"
  exit 1
fi

# Calculate shard using SHA256 hash (matching Terraform logic)
HASH=$(echo -n "$REPO_NAME" | shasum -a 256 | awk '{print $1}')

# Convert first 16 hex chars to decimal and mod by number of shards
SHARD=$(python3 -c "print(int('${HASH:0:16}', 16) % $NUM_SHARDS)")

echo "Repository: $REPO_NAME"
echo "Shard: $SHARD"
echo "Workspace: github-config-prod-<org>-shard-$SHARD"
