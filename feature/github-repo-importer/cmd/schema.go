package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

const (
	schemaOutDir       = ".schemas"
	schemaOutFile      = "repository-config.schema.json"
	teamsSchemaOutFile = "teams-config.schema.json"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for the repository and teams configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := "../../"
		if err := os.MkdirAll(fmt.Sprintf("%s/%s", projectRoot, schemaOutDir), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", schemaOutDir, err)
		}

		schemas := []struct {
			outFile string
			marshal func() ([]byte, error)
		}{
			{schemaOutFile, MarshalRepositoryConfigSchema},
			{teamsSchemaOutFile, MarshalTeamsConfigSchema},
		}

		for _, s := range schemas {
			outPath := filepath.Join(projectRoot, schemaOutDir, s.outFile)

			data, err := s.marshal()
			if err != nil {
				return err
			}
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				return fmt.Errorf("write schema file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Schema written to %s\n", outPath)
		}
		return nil
	},
}

// MarshalRepositoryConfigSchema returns the JSON-encoded repository config schema.
func MarshalRepositoryConfigSchema() ([]byte, error) {
	data, err := json.MarshalIndent(BuildRepositoryConfigSchema(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	return data, nil
}

// BuildRepositoryConfigSchema reflects the JSON schema for the repository
// configuration from the Go structs and applies the manual constraints that
// cannot be expressed through struct tags. It is the single source of truth
// shared by the `schema` and `validate` commands.
func BuildRepositoryConfigSchema() *jsonschema.Schema {
	{
		reflector := &jsonschema.Reflector{
			AllowAdditionalProperties: false,
			FieldNameTag:              "yaml",
		}

		schema := reflector.Reflect(&RepositoryWithExpansionConfig{})
		schema.Title = "Repository Configuration"
		schema.ID = jsonschema.ID(fmt.Sprintf("https://raw.githubusercontent.com/G-Research/github-terraformer/refs/heads/main/%s/%s", schemaOutDir, schemaOutFile))

		squashIf := &jsonschema.Schema{
			Properties: orderedmap.New[string, *jsonschema.Schema](),
		}
		squashIf.Properties.Set("allow_squash_merge", &jsonschema.Schema{Const: true})

		squashThen := &jsonschema.Schema{
			Properties: orderedmap.New[string, *jsonschema.Schema](),
		}
		squashThen.Properties.Set("squash_merge_commit_title", &jsonschema.Schema{
			Enum: []interface{}{"PR_TITLE", "COMMIT_OR_PR_TITLE"},
		})
		squashThen.Properties.Set("squash_merge_commit_message", &jsonschema.Schema{
			Enum: []interface{}{"PR_BODY", "COMMIT_MESSAGES", "BLANK"},
		})

		mergeIf := &jsonschema.Schema{
			Properties: orderedmap.New[string, *jsonschema.Schema](),
		}
		mergeIf.Properties.Set("allow_merge_commit", &jsonschema.Schema{Const: true})

		mergeThen := &jsonschema.Schema{
			Properties: orderedmap.New[string, *jsonschema.Schema](),
		}
		mergeThen.Properties.Set("merge_commit_title", &jsonschema.Schema{
			Enum: []interface{}{"PR_TITLE", "MERGE_MESSAGE"},
		})
		mergeThen.Properties.Set("merge_commit_message", &jsonschema.Schema{
			Enum: []interface{}{"PR_BODY", "PR_TITLE", "BLANK"},
		})

		schema.AllOf = []*jsonschema.Schema{
			{If: squashIf, Then: squashThen},
			{If: mergeIf, Then: mergeThen},
		}

		if schema.Definitions == nil {
			schema.Definitions = jsonschema.Definitions{}
		}

		ruleDef, ok := schema.Definitions["Rule"]
		if ok && ruleDef != nil {
			ruleDef.Not = &jsonschema.Schema{
				Required: []string{"branch_name_pattern", "tag_name_pattern"},
			}
		}

		repoDef, ok := schema.Definitions["RepositoryWithExpansionConfig"]
		if ok && repoDef != nil && repoDef.Properties != nil {
			for _, field := range []string{"admin_collaborators", "admin_teams"} {
				if prop, exists := repoDef.Properties.Get(field); exists {
					prop.Deprecated = true
				}
			}
		}

		pagesDef, ok := schema.Definitions["Pages"]
		if ok && pagesDef != nil {
			pagesIf := &jsonschema.Schema{
				Properties: orderedmap.New[string, *jsonschema.Schema](),
			}
			pagesIf.Properties.Set("build_type", &jsonschema.Schema{Const: "legacy"})
			pagesIf.Required = []string{"build_type"}
			pagesDef.AllOf = append(pagesDef.AllOf, &jsonschema.Schema{
				If: pagesIf,
				Then: &jsonschema.Schema{
					Required: []string{"branch"},
				},
			})
		}

		return schema
	}
}

// MarshalTeamsConfigSchema returns the JSON-encoded teams config schema.
func MarshalTeamsConfigSchema() ([]byte, error) {
	data, err := json.MarshalIndent(BuildTeamsConfigSchema(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	return data, nil
}

// BuildTeamsConfigSchema reflects the JSON schema for the organisation teams
// configuration (organisation/teams.yaml) from the Go structs. Unlike the
// repository config schema it needs no manual conditional constraints; the
// cross-record checks live in github.TeamsConfig.Validate instead.
func BuildTeamsConfigSchema() *jsonschema.Schema {
	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		FieldNameTag:              "yaml",
	}

	schema := reflector.Reflect(&github.TeamsConfig{})
	schema.Title = "Teams Configuration"
	schema.ID = jsonschema.ID(fmt.Sprintf("https://raw.githubusercontent.com/G-Research/github-terraformer/refs/heads/main/%s/%s", schemaOutDir, teamsSchemaOutFile))

	return schema
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
