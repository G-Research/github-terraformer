#!/usr/bin/env bash
# Generates `terraform state mv` commands to migrate rulesets from the old flat
# module path to the new per-repo for_each path:
#
#   FROM: module.rulesets.github_repository_ruleset.ruleset["<sha256>"]
#   TO:   module.rulesets["<repo>"].github_repository_ruleset.ruleset["<sha256>"]
#
# Usage:
#   bash generate-state-mv.sh            # prints commands to stdout
#   bash generate-state-mv.sh | sh       # runs commands against HCP state
#
# Run from the feature/github-repo-provisioning directory.

set -euo pipefail

YAML_DIRS=(
  "gcss_config/importer_tmp_dir"
  "gcss_config/repos"
)

for dir in "${YAML_DIRS[@]}"; do
  if [[ ! -d "$dir" ]]; then
    continue
  fi

  for yaml_file in "$dir"/*.yaml "$dir"/*.yml; do
    [[ -f "$yaml_file" ]] || continue

    repo=$(basename "$yaml_file" | sed 's/\.\(yaml\|yml\)$//')

    # Extract ruleset names using Python (available on macOS/Linux; avoids yq dependency)
    ruleset_names=$(python3 - "$yaml_file" <<'PYEOF'
import sys, yaml
with open(sys.argv[1]) as f:
    config = yaml.safe_load(f)
rulesets = config.get("rulesets") or []
for rs in rulesets:
    print(rs["name"])
PYEOF
)

    while IFS= read -r name; do
      [[ -z "$name" ]] && continue
      sha=$(printf '%s/%s' "$repo" "$name" | sha256sum | awk '{print $1}')
      echo "terraform state mv 'module.rulesets.github_repository_ruleset.ruleset[\"${sha}\"]' 'module.rulesets[\"${repo}\"].github_repository_ruleset.ruleset[\"${sha}\"]'"
    done <<< "$ruleset_names"
  done
done
