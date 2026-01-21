package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	inputDir  string
	outputDir string
)

var expandCmd = &cobra.Command{
	Use:   "expand",
	Short: "Expand YAML configuration files with default values",
	Long:  `Expand YAML configuration files by adding default fields and sorting keys for deterministic output.`,
	RunE:  runExpand,
}

func init() {
	rootCmd.AddCommand(expandCmd)

	expandCmd.Flags().StringVarP(&inputDir, "input-dir", "d", "", "Input directory path")
	expandCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory path")
}

func runExpand(cmd *cobra.Command, args []string) error {
	if inputDir != "" && outputDir == "" {
		return fmt.Errorf("--output-dir must be specified when using --input-dir")
	}

	return expandDirectory(inputDir, outputDir)
}

func expandFile(input, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	expanded, err := expandYAML(data)
	if err != nil {
		return fmt.Errorf("failed to expand YAML: %w", err)
	}

	if err := os.WriteFile(output, expanded, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Expanded: %s -> %s\n", input, output)
	return nil
}

func expandDirectory(inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		outPath := filepath.Join(outputDir, relPath)

		outDir := filepath.Dir(outPath)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", err)
		}

		return expandFile(path, outPath)
	})
}

func expandYAML(data []byte) ([]byte, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	expanded := expandRulesets(config)

	delete(expanded, "high_integrity")

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	var node yaml.Node
	if err := node.Encode(expanded); err != nil {
		return nil, fmt.Errorf("failed to encode to node: %w", err)
	}

	sortYAMLNode(&node)

	if err := encoder.Encode(&node); err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return buf.Bytes(), nil
}

func sortYAMLNode(node *yaml.Node) {
	if node.Kind == yaml.MappingNode {
		type pair struct {
			key   *yaml.Node
			value *yaml.Node
		}

		pairs := make([]pair, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			pairs[i/2] = pair{key: node.Content[i], value: node.Content[i+1]}
		}

		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].key.Value < pairs[j].key.Value
		})

		node.Content = make([]*yaml.Node, 0, len(pairs)*2)
		for _, p := range pairs {
			node.Content = append(node.Content, p.key, p.value)
			sortYAMLNode(p.value)
		}
	} else if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			sortYAMLNode(item)
		}
	}
}

func expandRulesets(config map[string]interface{}) map[string]interface{} {
	if highIntegrity, ok := config["high_integrity"].(map[string]interface{}); ok {
		if enabled, exists := highIntegrity["enabled"]; exists && enabled == true {
			rulesets, _ := config["rulesets"].([]interface{})

			defaultBranchProtectionRuleset := map[string]interface{}{
				"name":        "auto-generated via high-integrity - Protect main branch",
				"enforcement": "active",
				"target":      "branch",
				"conditions": map[string]interface{}{
					"ref_name": map[string]interface{}{
						"include": []interface{}{"~DEFAULT_BRANCH"},
					},
				},
				"rules": map[string]interface{}{
					"deletion":                true,
					"non_fast_forward":        true,
					"required_linear_history": true,
					"pull_request": map[string]interface{}{
						"required_approving_review_count":   1,
						"require_code_owner_review":         false,
						"dismiss_stale_reviews_on_push":     true,
						"require_last_push_approval":        true,
						"required_review_thread_resolution": false,
					},
				},
			}
			tagProtectionRuleset := map[string]interface{}{
				"name":        "auto-generated via high-integrity - Make tags immutable",
				"enforcement": "active",
				"target":      "tag",
				"conditions": map[string]interface{}{
					"ref_name": map[string]interface{}{
						"include": []interface{}{"~ALL"},
					},
				},
				"rules": map[string]interface{}{
					"non_fast_forward": true,
					"update":           true,
					"deletion":         true,
				},
			}
			rulesets = append(rulesets, defaultBranchProtectionRuleset)
			rulesets = append(rulesets, tagProtectionRuleset)
			config["rulesets"] = rulesets
		}
	}

	return config
}
