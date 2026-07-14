package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTeamsConfigSchema(t *testing.T) {
	schema := BuildTeamsConfigSchema()

	assert.Equal(t, "Teams Configuration", schema.Title)
	assert.True(t, strings.Contains(string(schema.ID), teamsSchemaOutFile), "schema $id should point at %s, got %s", teamsSchemaOutFile, schema.ID)

	teamsDef, ok := schema.Definitions["TeamsConfig"]
	assert.True(t, ok, "schema should define TeamsConfig")
	if ok {
		_, hasTeams := teamsDef.Properties.Get("teams")
		assert.True(t, hasTeams, "TeamsConfig should have a teams property")
	}

	teamDef, ok := schema.Definitions["Team"]
	assert.True(t, ok, "schema should define Team")
	if ok {
		assert.Equal(t, []string{"name"}, teamDef.Required)

		visibility, hasVisibility := teamDef.Properties.Get("visibility")
		assert.True(t, hasVisibility, "Team should have a visibility property")
		if hasVisibility {
			assert.Equal(t, []interface{}{"visible", "secret"}, visibility.Enum)
		}
	}
}

func TestBuildMembersConfigSchema(t *testing.T) {
	schema := BuildMembersConfigSchema()

	assert.Equal(t, "Members Configuration", schema.Title)
	assert.True(t, strings.Contains(string(schema.ID), membersSchemaOutFile), "schema $id should point at %s, got %s", membersSchemaOutFile, schema.ID)

	membersDef, ok := schema.Definitions["MembersConfig"]
	assert.True(t, ok, "schema should define MembersConfig")
	if ok {
		_, hasMembers := membersDef.Properties.Get("members")
		assert.True(t, hasMembers, "MembersConfig should have a members property")
	}

	memberDef, ok := schema.Definitions["Member"]
	assert.True(t, ok, "schema should define Member")
	if ok {
		assert.Equal(t, []string{"username"}, memberDef.Required)

		role, hasRole := memberDef.Properties.Get("role")
		assert.True(t, hasRole, "Member should have a role property")
		if hasRole {
			assert.Equal(t, []interface{}{"owner", "member"}, role.Enum)
		}

		teams, hasTeams := memberDef.Properties.Get("teams")
		assert.True(t, hasTeams, "Member should have a teams property")
		if hasTeams {
			assert.True(t, teams.UniqueItems, "teams should require unique items")
		}
	}
}

func TestBuildRepositoryConfigSchema(t *testing.T) {
	schema := BuildRepositoryConfigSchema()

	assert.Equal(t, "Repository Configuration", schema.Title)
	assert.True(t, strings.Contains(string(schema.ID), repositorySchemaOutFile), "schema $id should point at %s, got %s", repositorySchemaOutFile, schema.ID)
}
