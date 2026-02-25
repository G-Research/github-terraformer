#!/bin/bash
# Demonstrate that repos don't move between shards when new repos are added

calculate_shard() {
  local repo_name="$1"
  local hash=$(echo -n "$repo_name" | shasum -a 256 | awk '{print $1}')
  local shard=$(python3 -c "print(int('${hash:0:16}', 16) % 10)")
  echo "$shard"
}

echo "=========================================="
echo "Shard Stability Demonstration"
echo "=========================================="
echo ""
echo "Let's see what happens when we add new repos..."
echo ""

# Example existing repos
declare -a existing_repos=(
  "payment-service"
  "user-api"
  "notification-worker"
  "frontend-app"
  "backend-service"
)

echo "BEFORE: Existing repositories"
echo "------------------------------"
for repo in "${existing_repos[@]}"; do
  shard=$(calculate_shard "$repo")
  printf "%-25s → Shard %d\n" "$repo" "$shard"
done

echo ""
echo "=========================================="
echo ""

# New repos to add
declare -a new_repos=(
  "analytics-service"
  "metrics-collector"
  "auth-gateway"
)

echo "ADDING: New repositories"
echo "------------------------"
for repo in "${new_repos[@]}"; do
  shard=$(calculate_shard "$repo")
  printf "%-25s → Shard %d (NEW)\n" "$repo" "$shard"
done

echo ""
echo "=========================================="
echo ""

echo "AFTER: All repositories"
echo "-----------------------"
all_repos=("${existing_repos[@]}" "${new_repos[@]}")
for repo in "${all_repos[@]}"; do
  shard=$(calculate_shard "$repo")
  if [[ " ${existing_repos[@]} " =~ " ${repo} " ]]; then
    printf "%-25s → Shard %d\n" "$repo" "$shard"
  else
    printf "%-25s → Shard %d ✨ NEW\n" "$repo" "$shard"
  fi
done

echo ""
echo "=========================================="
echo "KEY OBSERVATIONS:"
echo "=========================================="
echo ""
echo "✅ Existing repos stay in their original shards"
echo "✅ New repos are assigned to shards based on their hash"
echo "✅ NO repos move between shards"
echo "✅ Each shard operates independently"
echo ""
echo "WORKFLOW IMPACT:"
echo "----------------"
echo "When you add 'analytics-service':"
echo "  1. PR detects changed file: gcss_config/repos/analytics-service.yaml"
echo "  2. Calculate shard: hash('analytics-service') % 10"
shard=$(calculate_shard "analytics-service")
echo "  3. Result: Shard $shard"
echo "  4. ONLY shard $shard runs Terraform (2-3 min)"
echo "  5. All other shards: SKIPPED"
echo ""
echo "=========================================="
