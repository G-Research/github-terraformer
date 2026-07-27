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

func TestValidateOrg_NoFilesSkips(t *testing.T) {
	dir := t.TempDir()

	out, err := runValidateOrgCmd(t, dir, "alice")

	assert.NoError(t, err)
	assert.Contains(t, out, "skipping validation")
}

func TestValidateOrg_MembersOnlyNoTeamsFile(t *testing.T) {
	dir := newOrgConfigDir(t, map[string]string{
		"members.yaml": "members:\n  - username: alice\n    role: owner\n",
	})

	out, err := runValidateOrgCmd(t, dir, "alice")

	assert.NoError(t, err)
	assert.Contains(t, out, "ok -- organisation config")
}
