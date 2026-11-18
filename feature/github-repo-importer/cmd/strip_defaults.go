package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var (
	stripDefaultsReposDir string
	stripDefaultsConfig   string
)

var stripDefaultsCmd = &cobra.Command{
	Use:   "strip-defaults",
	Short: "Remove scalar properties equal to their default from existing repo configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStripDefaults(stripDefaultsReposDir, stripDefaultsConfig)
	},
}

var topLevelKeyRe = regexp.MustCompile(`^([a-z_]+):`)

// runStripDefaults rewrites every repo YAML under reposDir, removing the
// top-level scalar lines whose value equals the default. It edits the raw text
// line-by-line rather than re-marshalling the document, so nested blocks
// (rulesets, environments, ...) keep their exact bytes and no zero-value fields
// are introduced.
func runStripDefaults(reposDir, defaultsPath string) error {
	defaults, err := github.LoadDefaults(defaultsPath)
	if err != nil {
		return err
	}
	scalarDefaults := defaults.ScalarDefaults()

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return fmt.Errorf("failed to read repos dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || (!strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml")) {
			continue
		}

		path := filepath.Join(reposDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", name, err)
		}

		var top map[string]any
		if err := yaml.Unmarshal(data, &top); err != nil {
			return fmt.Errorf("failed to parse %q: %w", name, err)
		}

		strip := map[string]bool{}
		for key, def := range scalarDefaults {
			if val, ok := top[key]; ok && val == def {
				strip[key] = true
			}
		}
		if len(strip) == 0 {
			continue
		}

		out := stripTopLevelKeys(data, strip)
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return fmt.Errorf("failed to write %q: %w", name, err)
		}
	}

	return nil
}

// stripTopLevelKeys removes top-level `key: value` lines whose key is in strip,
// leaving every other line byte-for-byte unchanged.
func stripTopLevelKeys(data []byte, strip map[string]bool) []byte {
	lines := bytes.Split(data, []byte("\n"))
	kept := lines[:0]
	for _, line := range lines {
		if m := topLevelKeyRe.FindSubmatch(line); m != nil && strip[string(m[1])] {
			continue
		}
		kept = append(kept, line)
	}
	return bytes.Join(kept, []byte("\n"))
}

func init() {
	rootCmd.AddCommand(stripDefaultsCmd)
	stripDefaultsCmd.Flags().StringVar(&stripDefaultsReposDir, "repos", "./repos", "Path to the repos directory to slim")
	stripDefaultsCmd.Flags().StringVar(&stripDefaultsConfig, "config", "./defaults.yaml", "Path to the defaults file")
}
