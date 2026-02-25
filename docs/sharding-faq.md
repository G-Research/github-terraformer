# Workspace Sharding - Frequently Asked Questions

## Repository Assignment

### Q: What happens when I add a new repository?

**A:** The new repository is assigned to a shard based on its name hash. Existing repositories **never move**.

**Example:**
```
# Before
payment-service → Shard 2 (30 repos in shard 2)

# Add notification-service
notification-service → Shard 5 (31 repos in shard 5 now)

# After
payment-service → Still Shard 2 ✅ (unchanged)
```

**Workflow impact:**
- Only the affected shard runs (e.g., shard 5)
- Runtime: 2-3 minutes
- All other shards: skipped

---

### Q: Do repositories ever move between shards?

**A:** No, never. Hash-based assignment is deterministic and permanent.

A repository name always produces the same hash, so it always goes to the same shard.

```python
# Always true
hash("my-repo") % 10 == hash("my-repo") % 10

# Therefore
shard("my-repo", today) == shard("my-repo", tomorrow)
```

**The only way a repo moves shards is if you rename it** (because the hash input changes).

---

### Q: What if I rename a repository?

**A:** Renaming changes the hash, so the repo will belong to a different shard.

**Example:**
```bash
# Old name
"user-service" → hash → Shard 3

# Renamed
"users-api" → hash → Shard 7 (different!)
```

**What to do:**
1. Rename repo in GitHub first
2. Rename YAML file: `user-service.yaml` → `users-api.yaml`
3. PR will trigger:
   - Shard 3: Delete old repo resources (or skip if already gone)
   - Shard 7: Import renamed repo
4. Both shards run (2-3 min each, in parallel)

**Note:** This is rare. Most repos are never renamed.

---

### Q: Will shards become unbalanced over time?

**A:** Yes, slightly. Hash distribution isn't perfect, but it's very good.

**Expected variance:**
- With 300 repos: 28-32 repos per shard (±7% variance)
- With 500 repos: 48-52 repos per shard (±4% variance)
- With 1000 repos: 98-102 repos per shard (±2% variance)

**Impact:**
- Worst-case shard: 3.5 minutes
- Best-case shard: 2.5 minutes
- Still much better than 20 minutes monolithic

**Rebalancing:**
- Generally not needed
- If one shard grows to 2x average, consider increasing shard count (10 → 12)
- Can be done without moving existing repos (see "Increasing Shard Count" below)

---

### Q: How do I know which shard a repository belongs to?

**A:** Use the helper script:

```bash
./scripts/which-shard.sh my-repo
# Output: Repository: my-repo
#         Shard: 5
#         Workspace: github-config-prod-<org>-shard-5
```

Or calculate it directly:
```bash
echo -n "my-repo" | shasum -a 256 | head -c 16 | xargs python3 -c "import sys; print(int(sys.argv[1], 16) % 10)"
```

Or in Terraform:
```hcl
parseint(sha256("my-repo"), 16) % 10
```

---

### Q: Can I manually assign a repo to a specific shard?

**A:** No, and you shouldn't need to.

Hash-based assignment ensures:
- Predictable placement
- Even distribution
- No manual maintenance
- No coordination needed

**If you really need to:**
You could modify the Terraform code to have a manual override map, but this defeats the purpose and adds complexity.

---

## Workflow Behavior

### Q: What happens when I modify an existing repo config?

**A:** Only the shard containing that repo runs.

**Example:**
```yaml
# You update: payment-service.yaml (add a branch protection rule)
```

**Workflow:**
1. Detect changed file: `gcss_config/repos/payment-service.yaml`
2. Calculate shard: `payment-service` → Shard 2
3. Run Terraform on Shard 2 only
4. Runtime: 2-3 minutes
5. Other shards: skipped

---

### Q: What if I modify multiple repos in different shards?

**A:** All affected shards run in parallel.

**Example:**
```
Changed files:
- gcss_config/repos/payment-service.yaml   → Shard 2
- gcss_config/repos/user-api.yaml          → Shard 6
- gcss_config/repos/notification-service.yaml → Shard 5
```

**Workflow:**
1. Calculate affected shards: [2, 5, 6]
2. Run all 3 shards in parallel:
   - Shard 2: 2-3 min
   - Shard 5: 2-3 min
   - Shard 6: 2-3 min
3. Total time: ~3 minutes (parallel execution)

**Key benefit:** Still much faster than 20 minutes monolithic!

---

### Q: What if I change app-list.yaml or core files?

**A:** Core workspace runs first, then all shards that have repos affected by the change.

**Workflow:**
1. Core workspace runs (updates team/app mappings)
2. Calculate affected shards (repos that use those teams/apps)
3. Run affected shards in parallel

**Conservative approach:**
If unsure which repos are affected, run all shards. Still only ~3 minutes with parallel execution.

---

### Q: What about the import workflow?

**A:** Import workflow needs updates to target the correct shard.

**Updated workflow:**
1. User triggers import for "new-repo"
2. Calculate shard: `hash("new-repo") % 10` → Shard 4
3. Generate YAML: `gcss_config/importer_tmp_dir/new-repo.yaml`
4. Create PR
5. On merge: Only Shard 4 runs Terraform import
6. Runtime: 2-3 minutes (instead of 20 minutes)

**Implementation note:**
The import workflow needs a small addition to calculate and store the target shard.

---

## Performance & Scaling

### Q: What's the performance improvement?

**Current (monolithic):**
- Single workspace
- 300+ repos, ~1500 resources
- Plan: 10 minutes
- Apply: 10 minutes
- Total: 20 minutes

**With sharding:**
- 10 shard workspaces + 1 core
- ~30 repos per shard, ~150 resources per shard
- Plan: 2-3 minutes (worst shard)
- Apply: 2-3 minutes (worst shard)
- Total: ~3-5 minutes

**Improvement:** 4-6x faster

**Best case (single repo change):**
- Only 1 shard runs: 2-3 minutes
- Improvement: 6-7x faster

---

### Q: What happens when we grow to 500+ repositories?

**Option 1: Keep 10 shards**
- ~50 repos per shard
- Plan/apply: 4-5 minutes per shard
- Still much better than monolithic (would be 30+ minutes)

**Option 2: Increase to 15 shards**
- ~33 repos per shard
- Plan/apply: 2-3 minutes per shard
- Same performance as before

**Option 3: Per-repo sharding**
- 500+ workspaces
- Plan/apply: 5-10 seconds per repo
- Maximum performance but more complex

**Recommendation:** Start with 10, increase to 15-20 if needed.

---

### Q: How does this affect HCP Terraform costs?

**Current:** 1 workspace

**Sharded:** 11 workspaces (10 shards + 1 core)

**Cost factors:**
- Workspace count: 11x more workspaces
- Run time per workspace: ~7x less
- Runs per month: Same (one run per PR)
- Total compute time: ~1.5x more (11 workspaces × 3 min vs 1 workspace × 20 min)

**Estimated cost increase:** 30-50%

**Cost optimization:**
- Only run affected shards (not all 10)
- Typical PR: 1-2 shards run
- Average cost increase: ~20-30%

**Cost vs. Speed tradeoff:**
- 30% more cost
- 6x faster feedback
- Better developer experience
- Worth it for most teams

---

## Increasing Shard Count

### Q: Can I increase from 10 to 15 shards later?

**A:** Yes, but it requires careful migration.

**Why it's complex:**
Changing the shard count changes the distribution:
```
10 shards: hash(repo) % 10
15 shards: hash(repo) % 15  ← Different result!
```

**Example:**
```
"payment-service"
  With 10 shards: hash % 10 = 2 → Shard 2
  With 15 shards: hash % 15 = 7 → Shard 7 (moved!)
```

**Migration approach:**

**Option A: Gradual migration**
1. Create 5 new shards (10-14)
2. Repos that would move from old shards to new shards:
   - Keep in old location (manual override)
   - Gradually migrate over time
3. New repos go to correct shard (based on % 15)

**Option B: One-time reshard**
1. Create 15 new shards with fresh state
2. Let them import all repos
3. Switch workflows
4. Decommission old 10 shards

**Recommendation:** Start with 10, increase only if needed. The cost of resharding is high.

---

## Migration & Rollback

### Q: Can I test sharding before fully migrating?

**A:** Yes! Use the parallel deployment strategy.

**Testing approach:**
1. Keep monolithic workspace running (production)
2. Deploy sharded workspaces (parallel test)
3. Run plans in both:
   - Monolithic: should show no changes
   - Sharded: should show imports only
4. Compare results: should be identical
5. Once validated: switch workflows
6. Decommission monolithic after 1-2 weeks

**Zero downtime, zero risk.**

---

### Q: How do I rollback if something goes wrong?

**A:** Very easily!

**Immediate rollback (minutes):**
1. Revert workflow change:
   ```bash
   git revert <commit-sha>
   git push
   ```
2. Workflows automatically use monolithic workspace
3. Done!

**Why it's safe:**
- Monolithic workspace kept as backup
- No state destroyed
- Sharded workspaces independent

**After rollback:**
- Investigate issue
- Fix in sharded code
- Test again
- Re-deploy when ready

---

## Edge Cases

### Q: What if two repos have a collision (same hash)?

**A:** Impossible. SHA256 collisions are astronomically rare.

SHA256 produces 2^256 possible values. For a collision:
- Probability: ~1 in 2^128 (1 in 340 trillion trillion trillion)
- Would need: 2^128 repos (more atoms than in the universe)

**In practice:** You will never see a collision.

**Even if you did:** They'd just both be in the same shard (no problem).

---

### Q: What about repos with dependencies on each other?

**A:** Each repo is managed independently, so no issue.

**Example:**
- `frontend-app` → Shard 3
- `backend-api` → Shard 7

These repos can be in different shards because:
- They don't share Terraform resources
- GitHub allows them to interact normally
- No Terraform dependencies between them

**Only Terraform resources matter, not application dependencies.**

---

### Q: What if GitHub API rate limits are hit?

**A:** Better with sharding!

**Monolithic:**
- Single workspace makes many API calls
- Higher chance of rate limiting

**Sharded:**
- Each shard makes fewer API calls
- Parallel execution spreads load
- Lower chance of rate limiting per shard

**If rate limited:**
- Only affects one shard
- Other shards unaffected
- Retry the failed shard

---

### Q: Can I run shards sequentially instead of parallel?

**A:** Yes, but you lose the performance benefit.

**Sequential execution:**
```yaml
strategy:
  matrix:
    shard_id: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
  max-parallel: 1  # Sequential
```

**Result:**
- 10 shards × 3 min each = 30 minutes
- Slower than monolithic (20 minutes)!

**Parallel execution:**
```yaml
strategy:
  matrix:
    shard_id: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
  max-parallel: 10  # Parallel
```

**Result:**
- 10 shards × 3 min (parallel) = 3 minutes
- 6x faster than monolithic!

**Recommendation:** Always use parallel execution.

---

## Best Practices

### Q: Should I use 10, 15, or 20 shards?

**Guidelines:**

| Repo Count | Recommended Shards | Repos/Shard | Shard Runtime |
|------------|-------------------|-------------|---------------|
| 50-150 | 5 shards | ~25 | 1-2 min |
| 150-400 | 10 shards | ~30 | 2-3 min |
| 400-800 | 15 shards | ~35 | 3-4 min |
| 800-1500 | 20 shards | ~50 | 4-5 min |
| 1500+ | Per-repo | 1 | 5-10 sec |

**Your case (300 repos):** 10 shards is perfect.

**Rule of thumb:** Target 20-40 repos per shard for optimal balance.

---

### Q: Should I use hash-based or range-based sharding?

**Hash-based (recommended):**
✅ Repos never move
✅ Even distribution
✅ No maintenance
✅ Deterministic
❌ Can't manually control placement

**Range-based (e.g., A-M, N-Z):**
✅ Intuitive
✅ Easy to understand
❌ Repos may need to move when rebalancing
❌ Uneven distribution (more repos start with 's' than 'z')
❌ Manual maintenance

**Recommendation:** Always use hash-based.

---

### Q: How often should I run the distribution analysis?

**Schedule:**
- Initially: After every 50 new repos
- Ongoing: Quarterly or semi-annually
- Trigger: If one shard becomes 2x the average

**Analysis:**
```bash
./scripts/analyze-shard-distribution.sh
```

**Action items:**
- If variance > 50%: Consider adding more shards
- If max shard > 60 repos: Consider rebalancing
- If runtime > 5 min: Optimize or increase shards

---

## Summary

**Key Takeaways:**

1. ✅ Repos never move between shards (hash-based assignment)
2. ✅ New repos automatically assigned to correct shard
3. ✅ Only affected shards run (fast feedback)
4. ✅ Easy to test and rollback
5. ✅ Scales well (300 → 500+ repos)
6. ⚠️ Slight cost increase (~30-50%)
7. ⚠️ More workspaces to manage (11 total)
8. ⚠️ Resharding is complex (avoid if possible)

**When to use sharding:**
- ✅ 150+ repositories
- ✅ 20+ minute Terraform runs
- ✅ Frequent PRs (many developers)
- ✅ Need fast feedback

**When NOT to use sharding:**
- ❌ < 100 repositories
- ❌ < 10 minute Terraform runs
- ❌ Infrequent changes
- ❌ Limited HCP Terraform budget

**Your case (300+ repos, 20 min runtime):** Perfect fit for sharding! 🎯
