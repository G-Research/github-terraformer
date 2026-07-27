package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOrgConfigDir creates a temporary config directory containing the given
// organisation/*.yaml files (name -> content). A nil/absent entry is skipped.
func newOrgConfigDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	orgDir := filepath.Join(dir, "organisation")
	require.NoError(t, os.MkdirAll(orgDir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(orgDir, name), []byte(content), 0o644))
	}
	return dir
}

func runValidateOrgCmd(t *testing.T, configDir, protectedOwners string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := runValidateOrg(cmd, configDir, protectedOwners, "", "")
	return out.String(), err
}

const validTeamsYAML = "teams:\n  - name: platform\n    visibility: visible\n  - name: security-core\n    visibility: secret\n"

func TestValidateOrg_ValidConfigPasses(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: alice\n    role: owner\n    teams:\n      - name: platform\n        role: maintainer\n  - username: bob\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "alice")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}

func TestValidateOrg_SchemaViolationCitesFileAndPath(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml": "teams:\n  - name: platform\n    visibility: banana\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, filepath.Join(dir, "organisation", "teams.yaml"))
	assert.Contains(t, out, "/teams/0/visibility")
}

func TestValidateOrg_DuplicateTeamNameRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml": "teams:\n  - name: platform\n    visibility: visible\n  - name: platform\n    visibility: visible\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `team "platform" is defined more than once`)
}

func TestValidateOrg_MemberReferencesUnknownTeam(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: alice\n    role: owner\n    teams:\n      - name: ghost\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `references team "ghost" which is not defined in teams.yaml`)
}

func TestValidateOrg_ProtectedOwnerRemovedRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: bob\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "alice")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "alice" is missing from members.yaml`)
}

func TestValidateOrg_ProtectedOwnerDemotedRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: alice\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "alice")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "alice" has role "member"`)
}

func TestValidateOrg_ProtectedOwnersCSVParsed(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	// bob is protected but absent; alice is protected and present as owner.
	out, err := runValidateOrgCmd(t, dir, " alice , bob ")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "bob" is missing from members.yaml`)
	assert.NotContains(t, out, `"alice"`)
}

func TestValidateOrg_NoFilesAndNoProtectedOwnersPasses(t *testing.T) {
	dir := t.TempDir()

	out, err := runValidateOrgCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}

// Terraform reads an absent members.yaml as an empty member list, so deleting
// the file plans the same org-wide member wipe as emptying it. Both must fail.
func TestValidateOrg_DeletedMembersFileStillEnforcesProtectedOwners(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml": validTeamsYAML,
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "gcss-bot" is missing from members.yaml`)
}

func TestValidateOrg_EmptyMembersListEnforcesProtectedOwners(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members: []\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "gcss-bot" is missing from members.yaml`)
}

// newStagedOrgConfigDir writes files to importer_tmp_dir/organisation/, where the
// bootstrap importer stages output that Terraform merges into the applied set.
func newStagedOrgConfigDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	stagedDir := filepath.Join(dir, "importer_tmp_dir", "organisation")
	require.NoError(t, os.MkdirAll(stagedDir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(stagedDir, name), []byte(content), 0o644))
	}
	return dir
}

func TestValidateOrg_StagedConfigIsSchemaValidated(t *testing.T) {
	dir := newStagedOrgConfigDir(t, map[string]string{
		"teams.yaml": "teams:\n  - name: platform\n    visibility: banana\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, filepath.Join("importer_tmp_dir", "organisation", "teams.yaml"))
	assert.Contains(t, out, "/teams/0/visibility")
}

func TestValidateOrg_StagedMembersCrossFileRulesEnforced(t *testing.T) {
	dir := newStagedOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: alice\n    role: owner\n    teams:\n      - name: ghost\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err)
	assert.Contains(t, out, `references team "ghost" which is not defined in teams.yaml`)
	assert.Contains(t, out, `protected owner "gcss-bot" is missing from members.yaml`)
}

// A staged member may legitimately reference a team that is already promoted, so
// team names from both locations must be visible to the reference check.
func TestValidateOrg_StagedMembersResolveAgainstPromotedTeams(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml": validTeamsYAML,
	})
	stagedDir := filepath.Join(dir, "importer_tmp_dir", "organisation")
	require.NoError(t, os.MkdirAll(stagedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stagedDir, "members.yaml"),
		[]byte("members:\n  - username: alice\n    role: owner\n    teams:\n      - name: platform\n"), 0o644))

	out, err := runValidateOrgCmd(t, dir, "alice")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}

// Terraform merges promoted over staged per username; the protected-owner check
// must see the merged result, not either file alone.
func TestValidateOrg_PromotedMemberOverridesStaged(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: member\n",
	})
	stagedDir := filepath.Join(dir, "importer_tmp_dir", "organisation")
	require.NoError(t, os.MkdirAll(stagedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stagedDir, "members.yaml"),
		[]byte("members:\n  - username: alice\n    role: owner\n"), 0o644))

	out, err := runValidateOrgCmd(t, dir, "alice")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "alice" has role "member"`)
}

func TestValidateOrg_MembersOnlyNoTeamsFile(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "alice")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}
