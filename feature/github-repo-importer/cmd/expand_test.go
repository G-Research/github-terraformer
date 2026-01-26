package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExpandYAML_NoHighIntegrity(t *testing.T) {
	input := `description: Test repository
visibility: public
default_branch: main`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(result, &output)
	require.NoError(t, err)

	_, hasHighIntegrity := output["high_integrity"]
	assert.False(t, hasHighIntegrity, "high_integrity field should not be present")

	rulesets, ok := output["rulesets"]
	if ok {
		assert.Nil(t, rulesets, "rulesets should be nil when high_integrity is not enabled")
	}
}

func TestExpandYAML_HighIntegrityEnabled(t *testing.T) {
	input := `description: High integrity repository
visibility: private
default_branch: main
high_integrity:
  enabled: true`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(result, &output)
	require.NoError(t, err)

	_, exists := output["high_integrity"]
	assert.False(t, exists, "high_integrity field should be removed from output")

	rulesets, ok := output["rulesets"].([]interface{})
	require.True(t, ok, "rulesets should be an array")
	require.Len(t, rulesets, 2, "should have exactly two rulesets")

	branchRuleset := findRulesetByName(rulesets, "auto-generated via high-integrity - Protect main branch")
	require.NotNil(t, branchRuleset, "branch protection ruleset should exist")
	assert.Equal(t, "active", branchRuleset["enforcement"])
	assert.Equal(t, "branch", branchRuleset["target"])

	conditions, ok := branchRuleset["conditions"].(map[string]interface{})
	require.True(t, ok)
	refName, ok := conditions["ref_name"].(map[string]interface{})
	require.True(t, ok)
	include, ok := refName["include"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, include, "~DEFAULT_BRANCH")

	rules, ok := branchRuleset["rules"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, rules["deletion"])
	assert.Equal(t, true, rules["non_fast_forward"])
	assert.Equal(t, true, rules["required_linear_history"])

	pullRequest, ok := rules["pull_request"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, pullRequest["required_approving_review_count"])
	assert.Equal(t, false, pullRequest["require_code_owner_review"])
	assert.Equal(t, true, pullRequest["dismiss_stale_reviews_on_push"])
	assert.Equal(t, true, pullRequest["require_last_push_approval"])
	assert.Equal(t, false, pullRequest["required_review_thread_resolution"])

	tagRuleset := findRulesetByName(rulesets, "auto-generated via high-integrity - Make tags immutable")
	require.NotNil(t, tagRuleset, "tag protection ruleset should exist")
	assert.Equal(t, "active", tagRuleset["enforcement"])
	assert.Equal(t, "tag", tagRuleset["target"])

	tagConditions, ok := tagRuleset["conditions"].(map[string]interface{})
	require.True(t, ok)
	tagRefName, ok := tagConditions["ref_name"].(map[string]interface{})
	require.True(t, ok)
	tagInclude, ok := tagRefName["include"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, tagInclude, "~ALL")

	tagRules, ok := tagRuleset["rules"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, tagRules["non_fast_forward"])
	assert.Equal(t, true, tagRules["update"])
	assert.Equal(t, true, tagRules["deletion"])
}

func TestExpandYAML_HighIntegrityDisabled(t *testing.T) {
	input := `
high_integrity:
  enabled: false`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(result, &output)
	require.NoError(t, err)

	_, hasHighIntegrity := output["high_integrity"]
	assert.False(t, hasHighIntegrity)

	rulesets, ok := output["rulesets"]
	if ok {
		assert.Nil(t, rulesets, "should not add rulesets when high_integrity is disabled")
	}
}

func TestExpandYAML_PreservesExistingRulesets(t *testing.T) {
	input := `
rulesets:
  - name: existing-ruleset
    enforcement: active
    target: branch
high_integrity:
  enabled: true`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(result, &output)
	require.NoError(t, err)

	rulesets, ok := output["rulesets"].([]interface{})
	require.True(t, ok)
	require.Len(t, rulesets, 3, "should have existing + 2 new rulesets")

	assert.NotNil(t, findRulesetByName(rulesets, "existing-ruleset"))
	assert.NotNil(t, findRulesetByName(rulesets, "auto-generated via high-integrity - Protect main branch"))
	assert.NotNil(t, findRulesetByName(rulesets, "auto-generated via high-integrity - Make tags immutable"))
}

func TestExpandYAML_AlphabeticalSorting(t *testing.T) {
	input := `visibility: public
archived: false
description: Test
default_branch: main`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	// Parse as string to check order
	lines := string(result)

	// Check that fields appear in alphabetical order
	archivedPos := findLinePosition(lines, "archived:")
	defaultBranchPos := findLinePosition(lines, "default_branch:")
	descriptionPos := findLinePosition(lines, "description:")
	visibilityPos := findLinePosition(lines, "visibility:")

	assert.True(t, archivedPos < defaultBranchPos, "archived should come before default_branch")
	assert.True(t, defaultBranchPos < descriptionPos, "default_branch should come before description")
	assert.True(t, descriptionPos < visibilityPos, "description should come before visibility")
}

func TestExpandYAML_InvalidYAML(t *testing.T) {
	input := `invalid: yaml: content:
  - missing
    quotes`

	_, err := expandYAML([]byte(input))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestExpandFile(t *testing.T) {
	tmpDir := t.TempDir()

	inputPath := filepath.Join(tmpDir, "input.yaml")
	inputContent := `
description: Test
high_integrity:
  enabled: true`
	err := os.WriteFile(inputPath, []byte(inputContent), 0644)
	require.NoError(t, err)

	outputPath := filepath.Join(tmpDir, "output.yaml")
	err = expandFile(inputPath, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	outputContent, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(outputContent, &output)
	require.NoError(t, err)

	_, hasHighIntegrity := output["high_integrity"]
	assert.False(t, hasHighIntegrity, "high_integrity should be removed")

	rulesets, ok := output["rulesets"].([]interface{})
	require.True(t, ok)
	assert.Len(t, rulesets, 2, "should have 2 rulesets")
}

func TestExpandDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)

	files := map[string]string{
		"repo1.yaml": `
high_integrity:
  enabled: true`,
		"repo2.yml": `
description: No expansion`,
		"subdir/repo3.yaml": `
visibility: public
high_integrity:
  enabled: true`,
		"readme.txt": "This should be ignored",
	}

	for path, content := range files {
		fullPath := filepath.Join(inputDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	err = expandDirectory(inputDir, outputDir)
	require.NoError(t, err)

	yamlFiles := []string{"repo1.yaml", "repo2.yml", "subdir/repo3.yaml"}
	for _, file := range yamlFiles {
		outputPath := filepath.Join(outputDir, file)
		_, err := os.Stat(outputPath)
		assert.NoError(t, err, "Output file should exist: %s", file)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		var output map[string]interface{}
		err = yaml.Unmarshal(content, &output)
		require.NoError(t, err)

		_, hasHighIntegrity := output["high_integrity"]
		assert.False(t, hasHighIntegrity, "high_integrity should be removed from: %s", file)
	}

	txtPath := filepath.Join(outputDir, "readme.txt")
	_, err = os.Stat(txtPath)
	assert.True(t, os.IsNotExist(err), "Non-YAML file should not be copied")
}

func TestExpandDirectory_SkipsOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := tmpDir
	outputDir := filepath.Join(tmpDir, "output")

	inputFile := filepath.Join(inputDir, "test.yaml")
	err := os.WriteFile(inputFile, []byte("high_integrity:\n  enabled: true"), 0644)
	require.NoError(t, err)

	err = expandDirectory(inputDir, outputDir)
	require.NoError(t, err)

	outputFile := filepath.Join(outputDir, "test.yaml")
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	nestedOutput := filepath.Join(outputDir, "output")
	_, err = os.Stat(nestedOutput)
	assert.True(t, os.IsNotExist(err), "Should not create nested output directory")
}

func TestExpandDirectory_CreatesSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	nestedPath := filepath.Join(inputDir, "level1", "level2", "level3")
	err := os.MkdirAll(nestedPath, 0755)
	require.NoError(t, err)

	filePath := filepath.Join(nestedPath, "deep.yaml")
	err = os.WriteFile(filePath, []byte("name: deep-repo"), 0644)
	require.NoError(t, err)

	err = expandDirectory(inputDir, outputDir)
	require.NoError(t, err)

	outputFilePath := filepath.Join(outputDir, "level1", "level2", "level3", "deep.yaml")
	_, err = os.Stat(outputFilePath)
	assert.NoError(t, err, "Nested output file should exist")
}

func TestExpandFile_NonExistentInput(t *testing.T) {
	tmpDir := t.TempDir()
	err := expandFile(
		filepath.Join(tmpDir, "nonexistent.yaml"),
		filepath.Join(tmpDir, "output.yaml"),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read input file")
}

func TestSortYAMLNode_NestedStructures(t *testing.T) {
	input := `zebra: last
alpha: first
nested:
  zebra: last
  alpha: first
array:
  - name: item2
  - name: item1`

	result, err := expandYAML([]byte(input))
	require.NoError(t, err)

	var output map[string]interface{}
	err = yaml.Unmarshal(result, &output)
	require.NoError(t, err)

	lines := string(result)
	alphaPos := findLinePosition(lines, "alpha:")
	nestedPos := findLinePosition(lines, "nested:")
	zebraPos := findLinePosition(lines, "zebra:")

	assert.True(t, alphaPos < nestedPos, "alpha should come before nested")
	assert.True(t, nestedPos < zebraPos, "nested should come before zebra")

	nested, ok := output["nested"].(map[string]interface{})
	require.True(t, ok)

	nestedKeys := make([]string, 0, len(nested))
	for k := range nested {
		nestedKeys = append(nestedKeys, k)
	}
	assert.Equal(t, "alpha", nestedKeys[0])
	assert.Equal(t, "zebra", nestedKeys[1])
}

func findRulesetByName(rulesets []interface{}, name string) map[string]interface{} {
	for _, rs := range rulesets {
		if rsMap, ok := rs.(map[string]interface{}); ok {
			if rsMap["name"] == name {
				return rsMap
			}
		}
	}
	return nil
}

func findLinePosition(content, searchStr string) int {
	lines := []byte(content)
	lineNum := 0
	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' {
			lineNum++
		}
		if i+len(searchStr) <= len(lines) {
			if string(lines[i:i+len(searchStr)]) == searchStr {
				return lineNum
			}
		}
	}
	return -1
}
