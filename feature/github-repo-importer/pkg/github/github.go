package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-github/v67/github"
	"github.com/shurcooL/githubv4"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/file"
)

var (
	v3client *github.Client
	v4client *githubv4.Client
)

func InitializeClients() {
	var err error
	v3client, v4client, err = CreateGitHubClient()
	if err != nil {
		panic(fmt.Sprintf("Error creating GitHub client: %v", err))
	}
}

func DecodeAppsList() (*AppsList, error) {
	appListFile, err := os.Open("./app-list.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to open app map file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}(appListFile)

	var appsList AppsList
	if err := yaml.NewDecoder(appListFile).Decode(&appsList); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &appsList, nil
}

func ImportRepo(repoName string) (*Repository, error) {
	fmt.Println("Importing repository: ", repoName)

	if !isValidRepoFormat(repoName) {
		return nil, errors.New("invalid repository format. Use owner/repo")
	}

	dumpManager, err := file.NewDumpManager(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new dump manager: %w", err)
	}

	repoNameSplit := strings.Split(repoName, "/")
	repo, r, err := v3client.Repositories.Get(context.Background(), repoNameSplit[0], repoNameSplit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo: %w (API Response: %s)", err, r.Status)
	}

	if err := dumpManager.WriteJSONFile("repository.json", repo); err != nil {
		fmt.Printf("failed to write repository.json: %v\n", err)
	}

	categorizedCollaborators, err := CategorizeCollaborators(v3client, repoNameSplit[0], repoNameSplit[1], dumpManager)
	if err != nil {
		return nil, fmt.Errorf("failed to categorize collaborators: %w", err)
	}

	categorizedTeams, err := CategorizeTeams(v3client, repoNameSplit[0], repoNameSplit[1], dumpManager)
	if err != nil {
		fmt.Printf("failed to categorize teams: %v\n", err)
	}

	pages, r, err := v3client.Repositories.GetPagesInfo(context.Background(), repoNameSplit[0], repoNameSplit[1])
	if err != nil {
		fmt.Printf("failed to get pages info: %v\n", err)
	}

	if err := dumpManager.WriteJSONFile("pages.json", pages); err != nil {
		fmt.Printf("failed to write pages.json: %v\n", err)
	}

	rulesets, r, err := v3client.Repositories.GetAllRulesets(context.Background(), repoNameSplit[0], repoNameSplit[1], false)
	if err != nil {
		if r.StatusCode == http.StatusForbidden {
			fmt.Printf("skipping rulesets due to insufficient permissions: %v\n", err)
		} else {
			return nil, fmt.Errorf("failed to get all rulesets: %v", err)
		}
	}

	var collectedRulesets []github.Ruleset
	for _, ruleset := range rulesets {
		rulesetById, _, _ := v3client.Repositories.GetRuleset(context.Background(), repoNameSplit[0], repoNameSplit[1], ruleset.GetID(), false)
		if err != nil {
			return nil, fmt.Errorf("failed to get rullset %v: %w", ruleset, err)
		}
		collectedRulesets = append(collectedRulesets, *rulesetById)
		filename := fmt.Sprintf("ruleset%d.json", rulesetById.GetID())
		if err := dumpManager.WriteJSONFile(filename, rulesetById); err != nil {
			fmt.Printf("failed to write json file %q: %v\n", filename, err)
		}
	}

	vulnerabilityAlertsEnabled, r, err := v3client.Repositories.GetVulnerabilityAlerts(context.Background(), repoNameSplit[0], repoNameSplit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vulnerability alerts: %v\n", err)
	}

	vars := map[string]interface{}{
		"owner": githubv4.String(repoNameSplit[0]),
		"name":  githubv4.String(repoNameSplit[1]),
	}

	var branchProtectionRulesGraphQLQuery BranchProtectionRulesGraphQLQuery
	err = v4client.Query(context.Background(), &branchProtectionRulesGraphQLQuery, vars)
	if err != nil {
		fmt.Printf("Failed to fetch branch protection rules: %v\n", err)
	}

	if err := dumpManager.WriteJSONFile("branch_protection_rules-graphql.json", branchProtectionRulesGraphQLQuery); err != nil {
		fmt.Printf("failed to write branch_protection_rules.json: %v\n", err)
	}

	orgTeams, res, err := v3client.Teams.ListTeams(context.Background(), repoNameSplit[0], &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to get org teams: %w (API Response: %s)", err, res.Status)
	}

	if err := dumpManager.WriteJSONFile("org_teams.json", orgTeams); err != nil {
		fmt.Printf("failed to write org_teams.json: %v\n", err)
	}

	appsList, err := DecodeAppsList()
	if err != nil {
		fmt.Printf("failed to decode apps list: %v\n", err)
	}

	roleActors := getRoleActors()
	teamActors := getTeamActors(orgTeams)
	appActors := getAppActors(appsList)

	resolvedRulesets, err := resolveRulesets(collectedRulesets, roleActors, teamActors, appActors)
	if err != nil {
		fmt.Printf("failed to resolve rulesets: %v\n", err)
	}

	return &Repository{
		Name:                       repo.GetName(),
		Owner:                      repo.GetOwner().GetLogin(),
		Description:                repo.Description,
		Visibility:                 repo.GetVisibility(),
		HomepageURL:                repo.Homepage,
		DefaultBranch:              repo.GetDefaultBranch(),
		HasIssues:                  repo.HasIssues,
		HasProjects:                repo.HasProjects,
		HasWiki:                    repo.HasWiki,
		HasDownloads:               repo.HasDownloads,
		AllowMergeCommit:           repo.AllowMergeCommit,
		AllowRebaseMerge:           repo.AllowRebaseMerge,
		AllowSquashMerge:           repo.AllowSquashMerge,
		AllowAutoMerge:             repo.AllowAutoMerge,
		AllowUpdateBranch:          repo.AllowUpdateBranch,
		SquashMergeCommitTitle:     repo.SquashMergeCommitTitle,
		SquashMergeCommitMessage:   repo.SquashMergeCommitMessage,
		MergeCommitTitle:           repo.MergeCommitTitle,
		MergeCommitMessage:         repo.MergeCommitMessage,
		WebCommitSignoffRequired:   repo.WebCommitSignoffRequired,
		DeleteBranchOnMerge:        repo.DeleteBranchOnMerge,
		IsTemplate:                 repo.IsTemplate,
		HasDiscussions:             repo.HasDiscussions,
		Archived:                   repo.Archived,
		Topics:                     repo.Topics,
		PullCollaborators:          categorizedCollaborators.Pull,
		TriageCollaborators:        categorizedCollaborators.Triage,
		PushCollaborators:          categorizedCollaborators.Push,
		MaintainCollaborators:      categorizedCollaborators.Maintain,
		AdminCollaborators:         categorizedCollaborators.Admin,
		PullTeams:                  categorizedTeams.Pull,
		TriageTeams:                categorizedTeams.Triage,
		PushTeams:                  categorizedTeams.Push,
		MaintainTeams:              categorizedTeams.Maintain,
		AdminTeams:                 categorizedTeams.Admin,
		LicenseTemplate:            repo.LicenseTemplate,
		GitignoreTemplate:          repo.GitignoreTemplate,
		Template:                   resolveRepositoryTemplate(repo),
		Pages:                      resolvePages(pages),
		Rulesets:                   resolvedRulesets,
		VulnerabilityAlertsEnabled: &vulnerabilityAlertsEnabled,
		BranchProtectionsV4:        resolveBranchProtectionsFromGraphQL(&branchProtectionRulesGraphQLQuery),
	}, nil
}

func getRoleActors() map[int64]string {
	return map[int64]string{
		BypassActorId_OrganizationAdminRole: BypassActorRoleName_OrganizationAdminRole,
		BypassActorId_RepositoryAdminRole:   BypassActorRoleName_RepositoryAdminRole,
		BypassActorId_MaintainRole:          BypassActorRoleName_MaintainRole,
		BypassActorId_WriteRole:             BypassActorRoleName_WriteRole,
	}
}

func getTeamActors(teams []*github.Team) map[int64]string {
	actors := make(map[int64]string)

	for _, team := range teams {
		if team.ID != nil && team.Slug != nil {
			actors[*team.ID] = "team/" + *team.Slug
		}
	}

	return actors
}

func getAppActors(apps *AppsList) map[int64]string {
	actors := make(map[int64]string)

	for _, app := range apps.Apps {
		actors[app.AppId] = "app/" + app.AppOwner + "/" + app.AppSlug
	}

	return actors
}

func ImportRepos(cfg Config) ([]*Repository, error) {
	reposToImport := cfg.SelectedRepos

	// If selectedRepos list has items, we don't fetch all repos via API, but jump to fetching one by one
	if len(reposToImport) == 0 {
		opts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{PerPage: *cfg.PageSize},
		}
		for {
			ghRepositories, r, err := v3client.Repositories.ListByOrg(context.Background(), os.Getenv("OWNER"), opts)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch repos: %w (API Response: %s)", err, r.Status)
			}

			for _, ghRepo := range ghRepositories {

				if slices.Contains(cfg.IgnoredRepos, *ghRepo.FullName) {
					fmt.Printf("skipping ignored repository %s\n", *ghRepo.FullName)
					continue
				}

				if ghRepo.GetArchived() {
					fmt.Printf("skipping archived repository %s\n", *ghRepo.FullName)
					continue
				}

				reposToImport = append(reposToImport, *ghRepo.FullName)
			}

			if r.NextPage == 0 {
				break
			}

			opts.Page = r.NextPage
		}
	}

	var importedRepos []*Repository
	for _, repoToImport := range reposToImport {
		repository, err := ImportRepo(repoToImport)
		if err != nil {
			return nil, fmt.Errorf("failed to import repository %s: %w", repository.Name, err)
		}
		importedRepos = append(importedRepos, repository)
	}
	return importedRepos, nil
}

func resolveBranchProtectionsFromGraphQL(query *BranchProtectionRulesGraphQLQuery) []*BranchProtectionV4 {
	if query == nil {
		return nil
	}

	var rules []*BranchProtectionV4

	for _, rule := range query.Repository.BranchProtectionRules.Nodes {
		var requiredPullRequestReviews *RequiredPullRequestReviews
		if anyTrue(rule.RequiredApprovingReviewCount != nil,
			rule.DismissesStaleReviews,
			rule.RequiresCodeOwnerReviews,
			len(resolveActors(rule.ReviewDismissalAllowances.Nodes)) > 0,
			rule.RestrictsReviewDismissals,
			rule.RequireLastPushApproval,
			len(resolveActors(rule.BypassPullRequestAllowances.Nodes)) > 0) {

			requiredPullRequestReviews = &RequiredPullRequestReviews{
				RequiredApprovingReviewCount: rule.RequiredApprovingReviewCount,
				DismissStaleReviews:          &rule.DismissesStaleReviews,
				RequireCodeOwnerReviews:      &rule.RequiresCodeOwnerReviews,
				DismissalRestrictions:        resolveActors(rule.ReviewDismissalAllowances.Nodes),
				RestrictDismissals:           &rule.RestrictsReviewDismissals,
				PullRequestBypassers:         resolveActors(rule.BypassPullRequestAllowances.Nodes),
				RequireLastPushApproval:      &rule.RequireLastPushApproval,
			}
		}

		var requiredStatusChecks *RequiredStatusChecksV4
		if rule.RequiresStatusChecks {
			requiredStatusChecks = &RequiredStatusChecksV4{
				Strict:   &rule.RequiresStrictStatusChecks,
				Contexts: resolveStatusChecksContexts(rule.RequiredStatusCheckContexts),
			}
		}

		rules = append(rules, &BranchProtectionV4{
			Pattern:                       string(rule.Pattern),
			AllowsDeletions:               &rule.AllowsDeletions,
			AllowsForcePushes:             &rule.AllowsForcePushes,
			ForcePushAllowances:           resolveActors(rule.BypassForcePushAllowances.Nodes),
			BlocksCreations:               &rule.BlocksCreations,
			EnforceAdmins:                 &rule.IsAdminEnforced,
			PushRestrictions:              resolveActors(rule.PushAllowances.Nodes),
			RequireConversationResolution: &rule.RequiresConversationResolution,
			RequireSignedCommits:          &rule.RequiresCommitSignatures,
			RequiredLinearHistory:         &rule.RequiresLinearHistory,
			RestrictsPushes:               &rule.RestrictsPushes,
			LockBranch:                    &rule.LockBranch,
			RequiredPullRequestReviews:    requiredPullRequestReviews,
			RequiredStatusChecks:          requiredStatusChecks,
		})
	}

	return rules
}

func anyTrue(first bool, others ...bool) bool {
	for _, b := range append(others, first) {
		if b {
			return true
		}
	}
	return false
}

func resolveActors(nodes []ActorWrapper) []string {
	if len(nodes) == 0 {
		return nil
	}

	var actors []string
	for _, node := range nodes {
		switch {
		case node.Actor.User.Name != "":
			actors = append(actors, "/"+string(node.Actor.User.Name))
		case node.Actor.Team.Name != "":
			actors = append(actors, string(node.Actor.Team.Name))
		case node.Actor.App.Name != "":
			actors = append(actors, "app/"+string(node.Actor.App.Name))
		}
	}
	return actors
}

func resolveStatusChecksContexts(contexts []githubv4.String) []string {
	if len(contexts) == 0 {
		return nil
	}

	var ctx []string
	for _, statusCheckContext := range contexts {
		ctx = append(ctx, string(statusCheckContext))
	}

	return ctx
}

func resolveRulesets(githubRulesets []github.Ruleset, roleActors map[int64]string, teamActors map[int64]string, appActors map[int64]string) ([]Ruleset, error) {
	var rulesets []Ruleset

	for _, githubRuleset := range githubRulesets {
		rules, err := convertRules(githubRuleset.Rules)
		if err != nil {
			return nil, fmt.Errorf("error occured while converting rules: %w", err)
		}

		rulesets = append(rulesets, Ruleset{
			ID:           githubRuleset.GetID(),
			Enforcement:  githubRuleset.Enforcement,
			Name:         githubRuleset.Name,
			Target:       githubRuleset.GetTarget(),
			BypassActors: convertBypassActors(githubRuleset.BypassActors, roleActors, teamActors, appActors),
			Conditions:   convertConditions(githubRuleset.Conditions),
			Rules:        rules,
		})
	}

	return rulesets, nil
}

func convertRules(ghRules []*github.RepositoryRule) (*Rule, error) {
	if len(ghRules) == 0 {
		return nil, nil
	}

	trueVal := true
	var rules Rule
	for _, r := range ghRules {
		switch r.Type {
		case RuleTypeRequiredLinearHistory:
			rules.RequiredLinearHistory = &trueVal

		case RuleTypePullRequest:
			prr, err := convertPullRequestRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.PullRequest = prr

		case RuleTypeRequiredStatusChecks:
			statusChecks, err := convertRequiredStatusChecks(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert required status checks %v: %v", r.Parameters, err)
			}
			rules.RequiredStatusChecks = statusChecks

		case RuleTypeDeletion:
			rules.Deletion = &trueVal

		case RuleTypeCreation:
			rules.Creation = &trueVal

		case RuleTypeNonFastForward:
			rules.NonFastForward = &trueVal

		case RuleRequiredSignatures:
			rules.RequiredSignatures = &trueVal

		case RuleUpdate:
			rules.Update = &trueVal
			updateAllowsFetchAndMerge, err := convertUpdateRequiresFetchAndMerge(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert required deployments %v: %v", r.Parameters, err)
			}
			rules.UpdateAllowsFetchAndMerge = updateAllowsFetchAndMerge

		case RuleRequiredDeployments:
			deployments, err := convertRequiredDeployments(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert required deployments %v: %v", r.Parameters, err)
			}
			rules.RequiredDeployments = deployments

		case RuleCommitMessagePattern:
			pattern, err := convertPatternRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.CommitMessagePattern = pattern

		case RuleCommitAuthorEmailPattern:
			pattern, err := convertPatternRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.CommitAuthorEmailPattern = pattern

		case RuleCommitterEmailPattern:
			pattern, err := convertPatternRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.CommitterEmailPattern = pattern

		case RuleBranchNamePattern:
			pattern, err := convertPatternRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.BranchNamePattern = pattern

		case RuleTagNamePattern:
			pattern, err := convertPatternRule(r.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pattern rule %v: %v", r.Parameters, err)
			}
			rules.TagNamePattern = pattern

		case RuleCodeScanning:
			rules.RequiredCodeScanning = convertRequiredCodeScanning(r.Parameters)

		default:
			// Handle unknown rule types
			fmt.Printf("Unknown rule type: %s\n", r.Type)
		}
	}
	return &rules, nil
}

func convertUpdateRequiresFetchAndMerge(parameters *json.RawMessage) (*bool, error) {
	if parameters == nil {
		return nil, nil
	}
	type Parameters struct {
		UpdateAllowsFetchAndMerge bool `json:"update_allows_fetch_and_merge"`
	}

	var params Parameters
	if err := json.Unmarshal(*parameters, &params); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal update requires fetch and merge: %v\n", err)
	}
	return &params.UpdateAllowsFetchAndMerge, nil
}

func convertPatternRule(pattern *json.RawMessage) (*PatternRule, error) {
	if pattern == nil {
		return nil, nil
	}
	var rule PatternRule
	if err := json.Unmarshal(*pattern, &rule); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal pattern rule: %v\n", err)
	}
	return &rule, nil
}

func convertPullRequestRule(pr *json.RawMessage) (*PullRequestRule, error) {
	if pr == nil {
		return nil, nil
	}
	var rule PullRequestRule
	err := json.Unmarshal(*pr, &rule)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal pull request rule: %v\n", err)
	}
	return &rule, nil
}

func convertRequiredDeployments(rd *json.RawMessage) (*RequiredDeployments, error) {
	if rd == nil {
		return nil, nil
	}
	var rule RequiredDeployments

	err := json.Unmarshal(*rd, &rule)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal required deployments: %v\n", err)
	}
	return &rule, nil
}

func convertRequiredStatusChecks(rsc *json.RawMessage) (*RequiredStatusChecks, error) {
	if rsc == nil {
		return nil, nil
	}

	var rule RequiredStatusChecks

	err := json.Unmarshal(*rsc, &rule)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal required status checks: %v\n", err)
	}
	return &rule, nil
}

func convertRequiredCodeScanning(rcs *json.RawMessage) *RequiredCodeScanning {
	if rcs == nil {
		return nil
	}

	var rule RequiredCodeScanning

	err := json.Unmarshal(*rcs, &rule)
	if err != nil {
		fmt.Printf("Failed to unmarshal required code scanning: %v\n", err)
	}
	return &rule
}

func convertConditions(ghConditions *github.RulesetConditions) *Conditions {
	if ghConditions == nil || ghConditions.RefName == nil {
		return nil
	}

	return &Conditions{
		RefName: RefNameCondition{
			Exclude: ghConditions.RefName.Exclude,
			Include: ghConditions.RefName.Include,
		},
	}
}

func convertBypassActors(ghActors []*github.BypassActor, roleActors map[int64]string, teamActors map[int64]string, appActors map[int64]string) []BypassActor {
	var result []BypassActor
	for _, actor := range ghActors {
		if actor == nil || actor.GetActorType() == BypassActorType_DeployKey {
			continue
		}

		name, err := resolveActorName(actor, roleActors, teamActors, appActors)
		if err != nil {
			fmt.Printf("Error resolving actor name for actor ID %d, and type %s: %v\n", actor.GetActorID(), actor.GetActorType(), err)
			continue
		}

		result = append(result, BypassActor{
			Name:       name,
			BypassMode: actor.BypassMode,
		})
	}
	return result
}

func resolveActorName(actor *github.BypassActor, roleActors map[int64]string, teamActors map[int64]string, appActors map[int64]string) (string, error) {
	switch actor.GetActorType() {
	case BypassActorType_RepositoryRole:
		roleActor := roleActors[actor.GetActorID()]
		if roleActor != "" {
			return roleActor, nil
		}
		return "", fmt.Errorf("unknown actor ID: %d, for actor type: %s", actor.GetActorID(), actor.GetActorType())
	case BypassActorType_OrganizationAdmin:
		return roleActors[BypassActorId_OrganizationAdminRole], nil
	case BypassActorType_Team:
		team := teamActors[actor.GetActorID()]
		if team != "" {
			return team, nil
		}
		return "", fmt.Errorf("unknown team ID: %d, for actor type: %s", actor.GetActorID(), actor.GetActorType())
	case BypassActorType_Integration:
		app := appActors[actor.GetActorID()]
		if app != "" {
			return app, nil
		}
		return "", fmt.Errorf("unknown app ID: %d, for actor type: %s", actor.GetActorID(), actor.GetActorType())
	default:
		return "", fmt.Errorf("unknown actor type: %s\n" + actor.GetActorType())
	}
}

func resolvePages(pages *github.Pages) *Pages {
	if pages != nil {
		return &Pages{
			CNAME:     pages.CNAME,
			Branch:    pages.GetSource().Branch,
			Path:      pages.GetSource().Path,
			BuildType: pages.BuildType,
		}
	}
	return nil
}

func resolveRepositoryTemplate(githubRepository *github.Repository) *RepositoryTemplate {
	if githubRepository.GetTemplateRepository() != nil {
		return &RepositoryTemplate{
			Owner:      githubRepository.GetTemplateRepository().GetOwner().GetLogin(),
			Repository: githubRepository.GetTemplateRepository().GetName(),
		}
	}
	return nil
}

func resolveVisibility(private bool) string {
	if private {
		return VisibilityPrivate
	}
	return VisibilityPublic
}

func isValidRepoFormat(repoName string) bool {
	return strings.Count(repoName, "/") == 1
}

type PermissionGroups struct {
	Pull     []string
	Triage   []string
	Push     []string
	Maintain []string
	Admin    []string
}

func CategorizeCollaborators(client *github.Client, owner, repo string, dumpManager *file.DumpManager) (*PermissionGroups, error) {
	var (
		pullCollaborators     []string
		triageCollaborators   []string
		pushCollaborators     []string
		maintainCollaborators []string
		adminCollaborators    []string
	)

	opts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Affiliation: "direct",
	}

	for {
		collaborators, resp, err := client.Repositories.ListCollaborators(context.Background(), owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch collaborators: %w", err)
		}

		filename := fmt.Sprintf("collaborators-page_%d.json", opts.Page+1)
		if err := dumpManager.WriteJSONFile(filename, collaborators); err != nil {
			fmt.Printf("failed to write %q: %v\n", filename, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}

		for _, collaborator := range collaborators {
			roleName := collaborator.GetRoleName()

			switch roleName {
			case PermissionRead:
				pullCollaborators = append(pullCollaborators, collaborator.GetLogin())
			case PermissionAdmin:
				adminCollaborators = append(adminCollaborators, collaborator.GetLogin())
			case PermissionTriage:
				triageCollaborators = append(triageCollaborators, collaborator.GetLogin())
			case PermissionPush, PermissionWrite:
				pushCollaborators = append(pushCollaborators, collaborator.GetLogin())
			case PermissionMaintain:
				maintainCollaborators = append(maintainCollaborators, collaborator.GetLogin())
			default:
				fmt.Printf("unknown role name: %s\n", roleName)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return &PermissionGroups{
		Pull:     pullCollaborators,
		Triage:   triageCollaborators,
		Push:     pushCollaborators,
		Maintain: maintainCollaborators,
		Admin:    adminCollaborators,
	}, nil
}

func CategorizeTeams(client *github.Client, owner, repo string, dumpManager *file.DumpManager) (*PermissionGroups, error) {
	var (
		pullTeams     []string
		triageTeams   []string
		pushTeams     []string
		maintainTeams []string
		adminTeams    []string
	)

	opts := &github.ListOptions{PerPage: 100}

	for {
		teams, resp, err := client.Repositories.ListTeams(context.Background(), owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}

		filename := fmt.Sprintf("teams-page_%d.json", opts.Page+1)
		if err := dumpManager.WriteJSONFile(filename, teams); err != nil {
			fmt.Printf("failed to write %q: %v\n", filename, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		for _, team := range teams {
			permission := team.GetPermission()
			switch permission {
			case PermissionPull:
				pullTeams = append(pullTeams, team.GetSlug())
			case PermissionAdmin:
				adminTeams = append(adminTeams, team.GetSlug())
			case PermissionTriage:
				triageTeams = append(triageTeams, team.GetSlug())
			case PermissionPush, PermissionWrite:
				pushTeams = append(pushTeams, team.GetSlug())
			case PermissionMaintain:
				maintainTeams = append(maintainTeams, team.GetSlug())
			default:
				fmt.Printf("unknown permission name: %s\n", permission)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return &PermissionGroups{
		Pull:     pullTeams,
		Triage:   triageTeams,
		Push:     pushTeams,
		Maintain: maintainTeams,
		Admin:    adminTeams,
	}, nil
}

func WriteRepositoryToYaml(repository *Repository) error {
	data, err := yaml.Marshal(repository)
	if err != nil {
		return fmt.Errorf("failed to marshal repository to YAML: %w", err)
	}

	configsBasePath := filepath.Join("./configs", repository.Owner)
	if err := os.MkdirAll(configsBasePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create base directories: %w", err)
	}

	if err := os.WriteFile(filepath.Join(configsBasePath, fmt.Sprintf("%s.yaml", repository.Name)), data, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write repository to YAML: %w", err)
	}

	return nil
}
