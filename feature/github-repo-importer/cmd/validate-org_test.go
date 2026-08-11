package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOrgConfigDir creates a temporary config directory containing the given
// organisation/*.yaml files (name -> content).
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
	return runValidateOrgCmdWithSchemas(t, configDir, protectedOwners, "", "")
}

func runValidateOrgCmdWithSchemas(t *testing.T, configDir, protectedOwners, fallbackTeams, fallbackMembers string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := runValidateOrg(cmd, configDir, protectedOwners, fallbackTeams, fallbackMembers)
	return out.String(), err
}

// writeGeneratedSchemas writes the generated schemas to disk so tests can exercise
// the file-based schema path.
func writeGeneratedSchemas(t *testing.T) (teamsPath, membersPath string) {
	t.Helper()
	dir := t.TempDir()

	teamsRaw, err := MarshalTeamsConfigSchema()
	require.NoError(t, err)
	teamsPath = filepath.Join(dir, "teams-config.schema.json")
	require.NoError(t, os.WriteFile(teamsPath, teamsRaw, 0o644))

	membersRaw, err := MarshalMembersConfigSchema()
	require.NoError(t, err)
	membersPath = filepath.Join(dir, "members-config.schema.json")
	require.NoError(t, os.WriteFile(membersPath, membersRaw, 0o644))

	return teamsPath, membersPath
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

func TestValidateOrg_DuplicateUsernameRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n  - username: alice\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `member "alice" is defined more than once`)
}

func TestValidateOrg_DuplicateInStagedFileNotHiddenByPromotedOverride(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})
	stagedDir := filepath.Join(dir, "importer_tmp_dir", "organisation")
	require.NoError(t, os.MkdirAll(stagedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stagedDir, "members.yaml"),
		[]byte("members:\n  - username: alice\n    role: owner\n  - username: alice\n    role: member\n"), 0o644))

	out, err := runValidateOrgCmd(t, dir, "alice")

	require.Error(t, err)
	assert.Contains(t, out, `member "alice" is defined more than once`)
	assert.Contains(t, out, filepath.Join("importer_tmp_dir", "organisation", "members.yaml"),
		"the duplicate should be attributed to the staged file that contains it")
}

func TestValidateOrg_BrokenTeamsFileStillEnforcesProtectedOwners(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   "teams: [oops\n",
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err)
	assert.Contains(t, out, "invalid YAML")
	assert.Contains(t, out, `protected owner "gcss-bot" is missing from members.yaml`)
}

func TestValidateOrg_UsernameCaseCollisionRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n  - username: Alice\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `member "Alice" collides with "alice"`)
}

func TestValidateOrg_TeamNameCaseCollisionRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml": "teams:\n  - name: platform\n    visibility: visible\n  - name: Platform\n    visibility: visible\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `team "Platform" collides with "platform"`)
}

func TestValidateOrg_ProtectedOwnerMatchedCaseInsensitively(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: GCSS-Bot\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}

func TestValidateOrg_ProtectedOwnerDemotionCaughtRegardlessOfCase(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: GCSS-Bot\n    role: member\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "gcss-bot" has role "member"`)
}

func TestValidateOrg_TeamReferenceMustMatchCaseExactly(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   "teams:\n  - name: platform\n    visibility: visible\n",
		"members.yaml": "members:\n  - username: alice\n    role: owner\n    teams:\n      - name: Platform\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, `references team "Platform" but teams.yaml defines it as "platform"`)
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

	out, err := runValidateOrgCmd(t, dir, " alice , bob ")

	require.Error(t, err)
	assert.Contains(t, out, `protected owner "bob" is missing from members.yaml`)
	assert.NotContains(t, out, "protected owner \"alice\"", "alice satisfies the guard and must not be reported")
	assert.Contains(t, out, "Enforcing 2 protected owner(s)")
}

func TestValidateOrg_WarnsWhenNoProtectedOwnersConfigured(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "WARNING: no protected owners configured")
}

func TestValidateOrg_FileBasedSchemasAreUsed(t *testing.T) {
	teamsSchema, membersSchema := writeGeneratedSchemas(t)
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   "teams:\n  - name: platform\n    visibility: banana\n",
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	out, err := runValidateOrgCmdWithSchemas(t, dir, "alice", teamsSchema, membersSchema)

	require.Error(t, err)
	assert.Contains(t, out, "/teams/0/visibility")
}

func TestValidateOrg_EmptyNamesRejected(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   "teams:\n  - name: \"\"\n    visibility: visible\n",
		"members.yaml": "members:\n  - username: \"\"\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, "/teams/0/name")
	assert.Contains(t, out, "/members/0/username")
}

func TestValidateOrg_BrokenTeamsFileDoesNotCascade(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"teams.yaml":   "teams: [oops\n",
		"members.yaml": "members:\n  - username: alice\n    role: owner\n    teams:\n      - name: platform\n",
	})

	out, err := runValidateOrgCmd(t, dir, "")

	require.Error(t, err)
	assert.Contains(t, out, "invalid YAML")
	assert.NotContains(t, out, `references team "platform"`, "team references must not be checked against an unparsed teams file")
	assert.Equal(t, 1, strings.Count(out, "invalid YAML"), "the parse failure should be reported once, not once per validation layer")
}

func TestValidateOrg_NoFilesAndNoProtectedOwnersPasses(t *testing.T) {
	dir := t.TempDir()

	out, err := runValidateOrgCmd(t, dir, "")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}

func TestValidateOrg_UnmanagedOrgDoesNotEnforceProtectedOwners(t *testing.T) {
	dir := t.TempDir()

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	assert.NoError(t, err, "an organisation with no config manages no memberships, so nothing can be removed")
	assert.Contains(t, out, "not under management yet")
	assert.Contains(t, out, "ok -- organisation config")
}

func TestValidateOrg_StagedBootstrapEnforcesProtectedOwners(t *testing.T) {
	dir := newStagedOrgConfigDir(t, map[string]string{
		"teams.yaml":   validTeamsYAML,
		"members.yaml": "members:\n  - username: someone-else\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "gcss-bot")

	require.Error(t, err, "enforcement must resume as soon as staged bootstrap output exists")
	assert.Contains(t, out, `protected owner "gcss-bot" is missing from members.yaml`)
}

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

// newStagedOrgConfigDir writes files to importer_tmp_dir/organisation/.
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
