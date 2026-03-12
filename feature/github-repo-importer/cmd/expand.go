package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	inputDir  string
	outputDir string
)

type ExpansionConfig struct {
	HighIntegrity *HighIntegrityConfig `yaml:"high_integrity,omitempty"`
}

type HighIntegrityConfig struct {
	Enabled bool `yaml:"enabled"`
}

type RepositoryWithExpansionConfig struct {
	github.Repository `yaml:",inline"`
	ExpansionConfig   `yaml:",inline"`
}

var expandCmd = &cobra.Command{
	Use:   "expand",
	Short: "Expand YAML configuration files with default values",
	Long:  `Expand YAML configuration files by adding default fields and sorting keys for deterministic output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return expandDirectory(inputDir, outputDir)
	},
}

func init() {
	rootCmd.AddCommand(expandCmd)

	expandCmd.Flags().StringVarP(&inputDir, "input-dir", "d", "", "Input directory path")
	expandCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory path")

	expandCmd.MarkPersistentFlagRequired("input-dir")
	expandCmd.MarkPersistentFlagRequired("output-dir")
}

func expandDirectory(inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	absInputDir, err := filepath.Abs(inputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute input path: %w", err)
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute output path: %w", err)
	}

	return filepath.Walk(absInputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if absPath == absOutputDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		relPath, err := filepath.Rel(absInputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		outPath := filepath.Join(absOutputDir, relPath)

		outDir := filepath.Dir(outPath)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", err)
		}

		return expandFile(path, outPath)
	})
}

func expandFile(input, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	expanded, err := expandYAML(data, input)
	if err != nil {
		return fmt.Errorf("failed to expand YAML: %w", err)
	}

	if err := os.WriteFile(output, expanded, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Expanded: %s -> %s\n", input, output)
	return nil
}

func warnDeprecatedFields(repo *RepositoryWithExpansionConfig, filename string) {
	if len(repo.AdminCollaborators) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %s: field 'admin_collaborators' will be deprecated in a future version\n", filename)
	}
	if len(repo.AdminTeams) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %s: field 'admin_teams' will be deprecated in a future version\n", filename)
	}
}

func expandYAML(data []byte, filename string) ([]byte, error) {
	var repoWithExpansion RepositoryWithExpansionConfig
	if err := yaml.Unmarshal(data, &repoWithExpansion); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	warnDeprecatedFields(&repoWithExpansion, filename)
	expandRulesets(&repoWithExpansion)
	repoWithExpansion.HighIntegrity = nil

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	var node yaml.Node
	if err := node.Encode(repoWithExpansion); err != nil {
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

func expandRulesets(repo *RepositoryWithExpansionConfig) {
	if repo.HighIntegrity != nil && repo.HighIntegrity.Enabled {
		defaultBranchProtectionRuleset := createDefaultBranchProtectionRuleset()
		repo.Rulesets = append(repo.Rulesets, defaultBranchProtectionRuleset)

		tagProtectionRuleset := createTagProtectionRuleset()
		repo.Rulesets = append(repo.Rulesets, tagProtectionRuleset)
	}
}

func createDefaultBranchProtectionRuleset() github.Ruleset {
	enforcement := "active"
	target := "branch"
	name := "auto-generated via high-integrity - Protect main branch"

	deletion := true
	nonFastForward := true
	requiredLinearHistory := true

	requiredApprovingReviewCount := 1
	requireCodeOwnerReview := false
	dismissStaleReviewsOnPush := true
	requireLastPushApproval := true
	requiredReviewThreadResolution := false

	return github.Ruleset{
		Name:        name,
		Enforcement: enforcement,
		Target:      target,
		Conditions: &github.Conditions{
			RefName: github.RefNameCondition{
				Include: []string{"~DEFAULT_BRANCH"},
			},
		},
		Rules: &github.Rule{
			Deletion:              &deletion,
			NonFastForward:        &nonFastForward,
			RequiredLinearHistory: &requiredLinearHistory,
			PullRequest: &github.PullRequestRule{
				RequiredApprovingReviewCount:   &requiredApprovingReviewCount,
				RequireCodeOwnerReview:         &requireCodeOwnerReview,
				DismissStaleReviewsOnPush:      &dismissStaleReviewsOnPush,
				RequireLastPushApproval:        &requireLastPushApproval,
				RequiredReviewThreadResolution: &requiredReviewThreadResolution,
			},
		},
	}
}

func createTagProtectionRuleset() github.Ruleset {
	enforcement := "active"
	target := "tag"
	name := "auto-generated via high-integrity - Make tags immutable"

	nonFastForward := true
	update := true
	deletion := true

	return github.Ruleset{
		Name:        name,
		Enforcement: enforcement,
		Target:      target,
		Conditions: &github.Conditions{
			RefName: github.RefNameCondition{
				Include: []string{"~ALL"},
			},
		},
		Rules: &github.Rule{
			NonFastForward: &nonFastForward,
			Update:         &update,
			Deletion:       &deletion,
		},
	}
}
