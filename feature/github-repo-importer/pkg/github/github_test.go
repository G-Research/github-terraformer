package github

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/google/go-github/v67/github"

	"github.com/stretchr/testify/assert"
)

// Mock execCommand
var execCommand = exec.Command

func TestIsValidRepoFormat(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		want     bool
	}{
		{
			name:     "valid repository format",
			repoName: "owner/repo",
			want:     true,
		},
		{
			name:     "invalid format - no slash",
			repoName: "ownerrepo",
			want:     false,
		},
		{
			name:     "invalid format - multiple slashes",
			repoName: "owner/repo/extra",
			want:     false,
		},
		{
			name:     "invalid format - empty string",
			repoName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRepoFormat(tt.repoName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveVisibility(t *testing.T) {
	tests := []struct {
		name    string
		private bool
		want    string
	}{
		{
			name:    "private repository",
			private: true,
			want:    VisibilityPrivate,
		},
		{
			name:    "public repository",
			private: false,
			want:    VisibilityPublic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVisibility(tt.private)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertBypassActors(t *testing.T) {
	var bypassModeAlways = "always"
	var bypassModePullRequest = "pull_request"

	teams := []*github.Team{
		{
			ID:   github.Int64(1000),
			Slug: github.String("dev-team"),
		},
		{
			ID:   github.Int64(1001),
			Slug: github.String("ops-team"),
		},
	}

	appsList := AppsList{
		Apps: []App{
			{
				AppId:    1111111,
				AppSlug:  "foo-bar",
				AppOwner: "test-org",
			},
			{
				AppId:    1111112,
				AppSlug:  "super-hero",
				AppOwner: "test-org",
			},
		},
	}

	roleActors := getRoleActors()
	teamActors := getTeamActors(teams)
	appActors := getAppActors(&appsList)

	var tests = []struct {
		name      string
		input     []*github.BypassActor
		expected  []BypassActor
		shouldLen int
	}{
		{
			name: "converts multiple actors",
			input: []*github.BypassActor{
				{
					ActorID:    github.Int64(BypassActorId_RepositoryAdminRole),
					ActorType:  github.String(BypassActorType_RepositoryRole),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					//ActorID: commented out because for Org Admin, ActorID is not set by Github API
					ActorType:  github.String(BypassActorType_OrganizationAdmin),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					ActorID:    github.Int64(BypassActorId_MaintainRole),
					ActorType:  github.String(BypassActorType_RepositoryRole),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					ActorID:    github.Int64(BypassActorId_WriteRole),
					ActorType:  github.String(BypassActorType_RepositoryRole),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					ActorID:    github.Int64(1000),
					ActorType:  github.String(BypassActorType_Team),
					BypassMode: github.String(bypassModePullRequest),
				},
				{
					ActorID:    github.Int64(1001),
					ActorType:  github.String(BypassActorType_Team),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					ActorID:    github.Int64(1111111),
					ActorType:  github.String(BypassActorType_Integration),
					BypassMode: github.String(bypassModePullRequest),
				},
				{
					ActorID:    github.Int64(1111112),
					ActorType:  github.String(BypassActorType_Integration),
					BypassMode: github.String(bypassModeAlways),
				},
				{
					//ActorID: commented out because for Org Admin, ActorID is not set by Github API
					ActorType:  github.String(BypassActorType_DeployKey),
					BypassMode: github.String(bypassModeAlways),
				},
			},
			expected: []BypassActor{
				{
					Name:       BypassActorRoleName_RepositoryAdminRole,
					BypassMode: &bypassModeAlways,
				},
				{
					Name:       BypassActorRoleName_OrganizationAdminRole,
					BypassMode: &bypassModeAlways,
				},
				{
					Name:       BypassActorRoleName_MaintainRole,
					BypassMode: &bypassModeAlways,
				},
				{
					Name:       BypassActorRoleName_WriteRole,
					BypassMode: &bypassModeAlways,
				},
				{
					Name:       "team/dev-team",
					BypassMode: &bypassModePullRequest,
				},
				{
					Name:       "team/ops-team",
					BypassMode: &bypassModeAlways,
				},
				{
					Name:       "app/test-org/foo-bar",
					BypassMode: &bypassModePullRequest,
				},
				{
					Name:       "app/test-org/super-hero",
					BypassMode: &bypassModeAlways,
				},
				// DeployKeys actor skipped - not supported
			},
			shouldLen: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBypassActors(tt.input, roleActors, teamActors, appActors)
			assert.Len(t, result, tt.shouldLen)
			for i, actor := range result {
				assert.Equal(t, tt.expected[i].Name, actor.Name)
				assert.Equal(t, tt.expected[i].BypassMode, actor.BypassMode)
			}
		})
	}
}

func TestImportRepo(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "invalid repo format",
			repoName:  "invalid-format",
			wantError: true,
			errorMsg:  "invalid repository format. Use owner/repo",
		},
		{
			name:      "empty repo name",
			repoName:  "",
			wantError: true,
			errorMsg:  "invalid repository format. Use owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := ImportRepo(tt.repoName, nil)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
			}
		})
	}
}

func TestResolvePages(t *testing.T) {
	tests := []struct {
		name     string
		input    *github.Pages
		expected *Pages
	}{
		{
			name: "legacy build type includes branch and path",
			input: &github.Pages{
				CNAME:     github.String("example.com"),
				BuildType: github.String("legacy"),
				Source: &github.PagesSource{
					Branch: github.String("gh-pages"),
					Path:   github.String("/docs"),
				},
			},
			expected: &Pages{
				CNAME:     github.String("example.com"),
				BuildType: github.String("legacy"),
				Branch:    github.String("gh-pages"),
				Path:      github.String("/docs"),
			},
		},
		{
			name: "workflow build type omits branch and path",
			input: &github.Pages{
				CNAME:     github.String("example.com"),
				BuildType: github.String("workflow"),
				Source:    &github.PagesSource{},
			},
			expected: &Pages{
				CNAME:     github.String("example.com"),
				BuildType: github.String("workflow"),
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePages(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPermissionGroupBuilder(t *testing.T) {
	var b permissionGroupBuilder

	// Added out of order, with duplicate roles, and using both "push" and its
	// alias "write" (the latter is what pending invitations report).
	b.add("zoe", PermissionPush)
	b.add("amy", PermissionWrite)
	b.add("mia", PermissionPush)
	b.add("bob", PermissionRead)
	b.add("carol", PermissionAdmin)
	b.add("dave", PermissionTriage)
	b.add("erin", PermissionMaintain)

	groups := b.build()

	// push + write collapse into one group, sorted deterministically.
	assert.Equal(t, []string{"amy", "mia", "zoe"}, groups.Push)
	assert.Equal(t, []string{"bob"}, groups.Pull)
	assert.Equal(t, []string{"carol"}, groups.Admin)
	assert.Equal(t, []string{"dave"}, groups.Triage)
	assert.Equal(t, []string{"erin"}, groups.Maintain)
}

func TestPermissionGroupBuilderEmpty(t *testing.T) {
	var b permissionGroupBuilder
	groups := b.build()

	assert.Nil(t, groups.Push)
	assert.Nil(t, groups.Pull)
	assert.Nil(t, groups.Admin)
	assert.Nil(t, groups.Triage)
	assert.Nil(t, groups.Maintain)
}

func TestConvertMergeQueue(t *testing.T) {
	raw := json.RawMessage(`{
		"check_response_timeout_minutes": 60,
		"grouping_strategy": "ALLGREEN",
		"max_entries_to_build": 5,
		"max_entries_to_merge": 5,
		"merge_method": "SQUASH",
		"min_entries_to_merge": 1,
		"min_entries_to_merge_wait_minutes": 5
	}`)

	mq, err := convertMergeQueue(&raw)
	assert.NoError(t, err)
	assert.NotNil(t, mq)
	assert.Equal(t, "SQUASH", mq.MergeMethod)
	assert.Equal(t, 5, *mq.BuildConcurrency)
	assert.Equal(t, 1, *mq.MinimumGroupSize)
	assert.Equal(t, 5, *mq.MaximumGroupSize)
	assert.Equal(t, 5, *mq.WaitTimeToMeetMinimumGroupSize)
	assert.Equal(t, 60, *mq.StatusCheckTimeout)
	// ALLGREEN maps to the "require all queue entries to pass" checkbox = true
	assert.Equal(t, true, *mq.RequireAllQueueEntriesToPassRequiredChecks)

	// HEADGREEN maps to false
	headGreen := json.RawMessage(`{"grouping_strategy": "HEADGREEN", "merge_method": "MERGE"}`)
	mq2, err := convertMergeQueue(&headGreen)
	assert.NoError(t, err)
	assert.Equal(t, false, *mq2.RequireAllQueueEntriesToPassRequiredChecks)

	nilMq, err := convertMergeQueue(nil)
	assert.NoError(t, err)
	assert.Nil(t, nilMq)
}
