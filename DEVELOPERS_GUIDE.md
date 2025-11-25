> [!IMPORTANT]
> Once you reach bypassers in rulesets or branch protection rules,
> remember that a Github Application can also be a bypasser.
> Currently we support only public Github Apps as bypassers,
> and such should be set as a bypasser in format `app/<app_name>`

## Repository Configuration

These are the primary configuration options for each repository.

**`name`** **does not exist**. The name of the YAML file is the name of the repository.

- **`default_branch`**: *(required, string)* Specifies the default branch for the repository. Example: `main`.

- **`description`**: *(optional, string)* A brief description of the repository.

- **`visibility`**: *(optional, string)* Defines the visibility of the repository. Possible values:
  - `public`
  - `private`

- **`homepage_url`**: *(optional, string)* The URL to the repository's homepage or website.

- **`has_issues`**: *(optional, boolean)* If `true`, the repository has issue tracking enabled.

- **`has_projects`**: *(optional, boolean)* If `true`, the repository has project boards enabled.

- **`has_wiki`**: *(optional, boolean)* If `true`, the repository has a wiki enabled.

- **`has_downloads`**: *(optional, boolean)* If `true`, the repository allows downloads (e.g., binary assets).

- **`allow_merge_commit`**: *(optional, boolean)* If `true`, merge commits are allowed.

- **`allow_rebase_merge`**: *(optional, boolean)* If `true`, rebase merging is allowed.

- **`allow_squash_merge`**: *(optional, boolean)* If `true`, squash merging is allowed.

- **`allow_auto_merge`**: *(optional, boolean)* If `true`, auto-merging is allowed when conditions are met.

- **`allow_update_branch`**: *(optional, boolean)* If `true`, contributors can update the branch from the default.

- **`squash_merge_commit_title`**: *(optional, string)* Defines the default title for squash merge commits. If `allow_squash_merge` is set to true, than this is set to one of:
  - `PR_TITLE`
  - `COMMIT_OR_PR_TITLE`

- **`squash_merge_commit_message`**: *(optional, string)* Defines the default commit message for squash merges. If `allow_squash_merge` is set to true, than this is set to one of:
  - `PR_BODY`
  - `COMMIT_MESSAGES`
  - `BLANK`

- **`merge_commit_title`**: *(optional, string)* Defines the default title for merge commits. If `allow_merge_commit` is set to true, than this is set to one of:
  - `PR_TITLE`
  - `MERGE_MESSAGE`

- **`merge_commit_message`**: *(optional, string)* Defines the default message for merge commits. If `allow_merge_commit` is set to true, than this is set to one of:
  - `PR_BODY`
  - `PR_TITLE`
  - `BLANK`

- **`web_commit_signoff_required`**: *(optional, boolean)* If `true`, commit signoff is required for web-based commits.

- **`delete_branch_on_merge`**: *(optional, boolean)* If `true`, the branch will be deleted after the merge.

- **`is_template`**: *(optional, boolean)* If `true`, the repository is a template repository.

- **`archived`**: *(optional, boolean)* If `true`, the repository is archived.

- **`has_discussions`**: *(optional, boolean)* If `true`, the repository has discussions enabled.

- **`topics`**: *(optional, string[])* A list of topics or tags to categorize the repository.

- **`pull_collaborators`**: *(optional, string[])* A list of users with pull access to the repository.

- **`triage_collaborators`**: *(optional, string[])* A list of users with triage access to the repository.

- **`push_collaborators`**: *(optional, string[])* A list of users with push access to the repository.

- **`maintain_collaborators`**: *(optional, string[])* A list of users with maintain access to the repository.

- **`admin_collaborators`**: *(optional, string[])* A list of users with admin access to the repository.

- **`pull_teams`**: *(optional, string[])* A list of teams with pull access to the repository.

- **`triage_teams`**: *(optional, string[])* A list of teams with triage access to the repository.

- **`push_teams`**: *(optional, string[])* A list of teams with push access to the repository.

- **`maintain_teams`**: *(optional, string[])* A list of teams with maintain access to the repository.

- **`admin_teams`**: *(optional, string[])* A list of teams with admin access to the repository.

- **`license_template`**: *(optional, string)* The license template to use for the repository. Use the [name of the template](https://github.com/github/choosealicense.com/tree/gh-pages/_licenses) without the extension. For example, "mit" or "mpl-2.0".

- **`gitignore_template`**: *(optional, string)* The gitignore template to use for the repository. Use the [name of the template](https://github.com/github/gitignore) without the extension. For example, "Haskel"

- **`template`**: *(optional, object [RepositoryTemplate](#template-configuration))* Configuration for creating a repository from a template.

- **`pages`**: *(optional, object [Pages](#pages-configuration))* Configuration for GitHub Pages.

- **`rulesets`**: *(optional, object[] [Ruleset](#ruleset-configuration))* Configuration for repository rulesets.

- **`vulnerability_alerts_enabled`**: *(optional, boolean)* If `true`, vulnerability alerts are enabled.

- **`branch_protections_v4`**: *(optional, object[] [BranchProtectionV4](#branch-protection-configuration-v4))* Configuration for branch protection rules.

## Template Configuration

Options for configuring a repository from a template.

- **`owner`**: *(required, string)* The owner of the template repository.

- **`repository`**: *(required, string)* The name of the template repository.

## Pages Configuration

Options for configuring GitHub Pages.

- **`cname`**: *(optional, string)* The custom domain for GitHub Pages.

- **`branch`**: *(required, string)* The branch to use for GitHub Pages.

- **`path`**: *(optional, string)* The directory path for GitHub Pages content.

- **`build_type`**: *(required, string)* The build type for GitHub Pages. Possible values:
  - `workflow` - For deploying pages by Github Actions workflow
  - `legacy` - For manual deployment using the `gh-pages` branch

## Ruleset Configuration

Options for configuring repository rulesets.

- **`enforcement`**: *(required, string)* The enforcement level of the ruleset. One of:
  - `disabled`
  - `active`
  - `evaluate` - currently only supported for owners of type organization

- **`name`**: *(required, string)* The name of the ruleset.

- **`rules`**: *(required, object [Rule](#rule-configuration))* The rules included in the ruleset.

- **`target`**: *(required, string)* The target of the ruleset.

- **`bypass_actors`**: *(optional, object[] [BypassActor](#bypass-actor-configuration))* Actors that can bypass the ruleset.

- **`conditions`**: *(optional, object [Conditions](#conditions-configuration))* Conditions for the ruleset.

- **`repository`**: *(optional, string)* The repository to which the ruleset applies.

## Rule Configuration

Options for configuring rules within a ruleset.

- **`branch_name_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for branch names. Conflicts with `tag_name_pattern`. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.

- **`tag_name_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for tag names. Conflicts with `branch_name_pattern`. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.

- **`commit_author_email_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for commit author emails. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.

- **`commit_message_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for commit messages. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.

- **`committer_email_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for committer emails. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.

- **`creation`**: *(optional, boolean)* If `true`, creation is allowed or required.

- **`deletion`**: *(optional, boolean)* If `true`, deletion is allowed or required.

- **`non_fast_forward`**: *(optional, boolean)* If `true`, non-fast-forward pushes are allowed or required.

- **`pull_request`**: *(optional, object [PullRequestRule](#pull-request-rule-configuration))* Configuration for pull request rules.

- **`required_deployments`**: *(optional, object [RequiredDeployments](#required-deployments-configuration))* Configuration for required deployments.

- **`required_linear_history`**: *(optional, boolean)* If `true`, a linear commit history is required.

- **`required_signatures`**: *(optional, boolean)* If `true`, commit signatures are required.

- **`required_status_checks`**: *(optional, object [RequiredStatusChecks](#required-status-checks-configuration))* Configuration for required status checks.

- **`tag_name_pattern`**: *(optional, object [PatternRule](#pattern-rule-configuration))* Pattern rule for tag names.

- **`required_code_scanning`**: *(optional, object [RequiredCodeScanning](#required-code-scanning-configuration))* Configuration for required code scanning.

- **`update`**: *(optional, boolean)* If `true`, updates are allowed or required.

- **`update_allows_fetch_and_merge`**: *(optional, boolean)* If `true`, fetch and merge updates are allowed.

## Pattern Rule Configuration

Options for configuring pattern rules.

- **`operator`**: *(required, string)* The operator for the pattern rule. One of:
  - `starts_with`
  - `ends_with`
  - `contains`
  - `regex`

- **`pattern`**: *(required, string)* The pattern to match.

- **`name`**: *(optional, string)* The name of the pattern rule.

- **`negate`**: *(optional, boolean)* If `true`, the pattern match is negated.

## Pull Request Rule Configuration

Options for configuring pull request rules.

- **`dismiss_stale_reviews_on_push`**: *(optional, boolean)* If `true`, stale reviews are dismissed on push.

- **`require_code_owner_review`**: *(optional, boolean)* If `true`, code owner review is required.

- **`require_last_push_approval`**: *(optional, boolean)* If `true`, approval of the last push is required.

- **`required_approving_review_count`**: *(optional, integer)* The number of required approving reviews.

- **`required_review_thread_resolution`**: *(optional, boolean)* If `true`, resolution of review threads is required.

## Required Deployments Configuration

Options for configuring required deployments.

- **`required_deployment_environments`**: *(required, string[], `minItems=1)* A list of required deployment environments.

## Required Status Checks Configuration

Options for configuring required status checks.

- **`required_check`**: *(required, object[] [RequiredCheck](#required-check-configuration), `minItems=1`)* A list of required status checks.

- **`strict_required_status_checks_policy`**: *(optional, boolean)* If `true`, strict status check policy is enforced.

## Required Check Configuration

Options for configuring a required status check.

- **`context`**: *(required, string)* The context of the status check.

- **`source`**: *(required, string)* Name of required check source. Usually an integration (Github app) name. Format `app/<app-owner>/<app-slug>`, both can be obtained from `app-list.yaml`

## Required Code Scanning Configuration

Options for configuring required code scanning.

- **`required_code_scanning_tool`**: *(required, object[] [RequiredCodeScanningTool](#required-code-scanning-tool-configuration), `minItems=1`)* A list of required code scanning tools.

## Required Code Scanning Tool Configuration

Options for configuring a required code scanning tool.

- **`alerts_threshold`**: *(required, string)* The threshold for alerts. One of:
  - `none`
  - `errors`
  - `errors_and_warnings`
  - `all`

- **`security_alerts_threshold`**: *(required, string)* The threshold for security alerts. One ofL
  - `none`
  - `critical`
  - `high_or_higher`
  - `medium_or_higher`
  - `all`

- **`tool`**: *(optional, string)* The name of the code scanning tool.

## Bypass Actor Configuration

Options for configuring actors that can bypass rules.

- **`name`**: *(required, string)* The name of the actor. Can be an actor role, a team or an integration (Github app). Teams should be prefixed with `team/<team-slug>`. Apps are prefixed with `app/<app-owner>/<app-slug>`. If an actor role, then of:
  - `repository-admin-role`
  - `organization-admin-role`
  - `maintain-role`
  - `write-role`

- **`bypass_mode`**: *(optional, enum)* The bypass mode for the actor. One of:
  - `always`
  - `pull_request`

## Conditions Configuration

Options for configuring conditions for rulesets.

- **`ref_name`**: *(required, object [RefNameCondition](#reference-name-condition-configuration))* Reference name conditions.

## Reference Name Condition Configuration

Options for configuring reference name conditions.

- **`exclude`**: *(required, string[], `minItems=1`)* A list of reference names to exclude.

- **`include`**: *(required, string[], `minItems=1`)* A list of reference names to include. Also accepts `~DEFAULT_BRANCH` to include the default branch or `~ALL` to include all branches.

## Branch Protection Configuration (V4)

Options for configuring branch protection rules.

- **`pattern`**: *(required, string)* The pattern for branches to protect.

- **`allows_deletions`**: *(optional, boolean)* If `true`, branch deletion is allowed.

- **`allows_force_pushes`**: *(optional, boolean)* If `true`, force pushes are allowed. Set to `false` if `force_push_bypassers` is set.

- **`force_push_bypassers`**: *(optional, string[])* A list of users or teams allowed to force push. If set, then `allows_force_pushes` must be set to `false`.

- **`blocks_creations`**: *(optional, boolean)* If `true`, branch creation is blocked.

- **`enforce_admins`**: *(optional, boolean)* If `true`, admins are also subject to branch protection.

- **`push_restrictions`**: *(optional, string[])* A list of users or teams allowed to push.

- **`require_conversation_resolution`**: *(optional, boolean)* If `true`, resolution of conversations is required.

- **`require_signed_commits`**: *(optional, boolean)* If `true`, signed commits are required.

- **`required_linear_history`**: *(optional, boolean)* If `true`, a linear commit history is required.

- **`required_pull_request_reviews`**: *(optional, object [RequiredPullRequestReviews](#required-pull-request-reviews-configuration))* Configuration for required pull request reviews.

- **`required_status_checks`**: *(optional, object [RequiredStatusChecksV4](#required-status-checks-configuration-v4))* Configuration for required status checks.

- **`restricts_pushes`**: *(optional, boolean)* If `true`, pushes are restricted.

- **`lock_branch`**: *(optional, boolean)* If `true`, the branch is locked.

## Required Pull Request Reviews Configuration

Options for configuring required pull request reviews.

- **`required_approving_review_count`**: *(optional, integer)* The number of required approving reviews. Must be between 0 and 6

- **`dismiss_stale_reviews`**: *(optional, boolean)* If `true`, stale reviews are dismissed.

- **`require_code_owner_reviews`**: *(optional, boolean)* If `true`, code owner reviews are required.

- **`dismissal_restrictions`**: *(optional, string[])* A list of users or teams allowed to dismiss reviews.

- **`restrict_dismissals`**: *(optional, boolean)* If `true`, review dismissals are restricted.

- **`pull_request_bypassers`**: *(optional, string[])* A list of users or teams allowed to bypass pull request requirements.

- **`require_last_push_approval`**: *(optional, boolean)* If `true`, approval of the last push is required.

## Required Status Checks Configuration (V4)

Options for configuring required status checks in V4.

- **`strict`**: *(optional, boolean)* If `true`, strict status checks are enforced.

- **`contexts`**: *(optional, string[])* A list of required status check contexts.