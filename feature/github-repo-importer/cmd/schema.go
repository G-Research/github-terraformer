package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for the repository config",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := "../../"
		outDir := ".schemas"
		outFile := "repository-config.schema.json"
		if err := os.MkdirAll(fmt.Sprintf("%s/%s", projectRoot, outDir), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", outDir, err)
		}

		outPath := filepath.Join(projectRoot, outDir, outFile)
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create schema file: %w", err)
		}
		defer f.Close()

		reflector := &jsonschema.Reflector{
			AllowAdditionalProperties: false,
			FieldNameTag:              "yaml",
		}

		schema := reflector.Reflect(&github.Repository{})
		schema.Title = "Repository Configuration"
		schema.ID = jsonschema.ID(fmt.Sprintf("https://raw.githubusercontent.com/G-Research/github-terraformer/refs/heads/main/%s/%s", outDir, outFile))

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

		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal schema: %w", err)
		}
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("write schema: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Schema written to %s\n", outPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
