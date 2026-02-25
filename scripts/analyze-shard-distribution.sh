#!/bin/bash
# Analyze how repositories would be distributed across shards

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$SCRIPT_DIR/../feature/github-repo-provisioning/gcss_config/repos"
NUM_SHARDS=10

echo "========================================"
echo "Shard Distribution Analysis"
echo "========================================"
echo ""

# Check if repo directory exists
if [ ! -d "$REPO_DIR" ]; then
  echo "Error: Repository directory not found: $REPO_DIR"
  echo "Please run this script from the github-terraformer root directory"
  exit 1
fi

# Find all YAML files
REPO_FILES=$(find "$REPO_DIR" -name "*.yaml" -o -name "*.yml" 2>/dev/null)

if [ -z "$REPO_FILES" ]; then
  echo "No repository YAML files found in $REPO_DIR"
  exit 1
fi

# Count total repos
TOTAL_REPOS=$(echo "$REPO_FILES" | wc -l | tr -d ' ')
echo "Total repositories: $TOTAL_REPOS"
echo "Target shards: $NUM_SHARDS"
echo "Expected per shard: $((TOTAL_REPOS / NUM_SHARDS))"
echo ""

# Initialize shard counters
declare -A shard_counts
declare -A shard_repos
for i in $(seq 0 $((NUM_SHARDS - 1))); do
  shard_counts[$i]=0
  shard_repos[$i]=""
done

# Calculate shard for each repo
while IFS= read -r file; do
  # Extract repo name from filename
  REPO_NAME=$(basename "$file" | sed 's/\.ya?ml$//')

  # Calculate shard using SHA256 hash (matching Terraform logic)
  HASH=$(echo -n "$REPO_NAME" | shasum -a 256 | awk '{print $1}')

  # Convert first 16 hex chars to decimal and mod by number of shards
  # Use Python for accurate large number arithmetic
  SHARD=$(python3 -c "print(int('${HASH:0:16}', 16) % $NUM_SHARDS)")

  # Increment counter
  shard_counts[$SHARD]=$((shard_counts[$SHARD] + 1))

  # Store repo name
  if [ -z "${shard_repos[$SHARD]}" ]; then
    shard_repos[$SHARD]="$REPO_NAME"
  else
    shard_repos[$SHARD]="${shard_repos[$SHARD]},$REPO_NAME"
  fi
done <<< "$REPO_FILES"

# Display results
echo "========================================"
echo "Distribution by Shard:"
echo "========================================"
for i in $(seq 0 $((NUM_SHARDS - 1))); do
  count=${shard_counts[$i]}
  percentage=$(python3 -c "print(f'{($count / $TOTAL_REPOS * 100):.1f}')")

  # Create bar chart
  bar_length=$((count / 3))  # Scale for display
  bar=$(printf '█%.0s' $(seq 1 $bar_length))

  printf "Shard %d: %3d repos (%5s%%) %s\n" $i $count $percentage "$bar"
done

echo ""
echo "========================================"
echo "Statistics:"
echo "========================================"

# Calculate min, max, avg
min_count=${shard_counts[0]}
max_count=${shard_counts[0]}
total_count=0

for i in $(seq 0 $((NUM_SHARDS - 1))); do
  count=${shard_counts[$i]}
  total_count=$((total_count + count))

  if [ $count -lt $min_count ]; then
    min_count=$count
  fi

  if [ $count -gt $max_count ]; then
    max_count=$count
  fi
done

avg_count=$((total_count / NUM_SHARDS))
variance=$((max_count - min_count))

echo "Minimum: $min_count repos"
echo "Maximum: $max_count repos"
echo "Average: $avg_count repos"
echo "Variance: $variance repos ($(python3 -c "print(f'{($variance / $avg_count * 100):.1f}')")% of average)"
echo ""

# Distribution quality assessment
if [ $variance -le $((avg_count / 10)) ]; then
  echo "✅ Distribution quality: EXCELLENT (variance < 10% of average)"
elif [ $variance -le $((avg_count / 5)) ]; then
  echo "✅ Distribution quality: GOOD (variance < 20% of average)"
elif [ $variance -le $((avg_count / 3)) ]; then
  echo "⚠️  Distribution quality: FAIR (variance < 33% of average)"
else
  echo "❌ Distribution quality: POOR (high variance)"
  echo "   Consider adjusting number of shards or using different hash function"
fi

echo ""
echo "========================================"
echo "Estimated Performance Impact:"
echo "========================================"

# Estimate time savings
CURRENT_TIME=20  # minutes
EST_SHARD_TIME=$(python3 -c "print(f'{($max_count / $TOTAL_REPOS * $CURRENT_TIME):.1f}')")
SPEEDUP=$(python3 -c "print(f'{($CURRENT_TIME / float($EST_SHARD_TIME)):.1f}')")

echo "Current monolithic time: ${CURRENT_TIME} minutes"
echo "Estimated shard time: ${EST_SHARD_TIME} minutes (worst case shard)"
echo "Estimated speedup: ${SPEEDUP}x"
echo ""
echo "Note: With parallel execution, all shards run simultaneously,"
echo "so total time ≈ slowest shard time = ${EST_SHARD_TIME} minutes"

echo ""
echo "========================================"
echo "Workspace Names:"
echo "========================================"
echo "Core:    github-config-prod-<org>-core"
for i in $(seq 0 $((NUM_SHARDS - 1))); do
  echo "Shard $i: github-config-prod-<org>-shard-$i"
done

echo ""
echo "========================================"
echo "Analysis complete!"
echo "========================================"
echo ""
