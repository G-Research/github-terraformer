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

func TestBuildRepositoryConfigSchema(t *testing.T) {
	schema := BuildRepositoryConfigSchema()

	assert.Equal(t, "Repository Configuration", schema.Title)
	assert.True(t, strings.Contains(string(schema.ID), schemaOutFile), "schema $id should point at %s, got %s", schemaOutFile, schema.ID)
}
