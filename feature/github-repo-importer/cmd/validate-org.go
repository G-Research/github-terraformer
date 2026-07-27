package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

// orgConfigDir holds the promoted organisation config; stagedOrgConfigDir holds
// the importer's not-yet-promoted output. Terraform merges both, so both are
// validated.
const (
	orgConfigDir       = "organisation"
	stagedOrgConfigDir = "importer_tmp_dir/organisation"
)

var (
	validateOrgConfigDir       string
	validateOrgProtectedOwners string
	validateOrgFallbackTeams   string
	validateOrgFallbackMembers string
)

var validateOrgCmd = &cobra.Command{
	Use:   "validate-org",
	Short: "Validate organisation teams.yaml and members.yaml before Terraform runs",
	Long: `ValidateOrg validates the organisation teams.yaml and members.yaml in the
config directory, failing fast before Terraform runs. Files are read from both
organisation/ and importer_tmp_dir/organisation/ and merged the same way the
Terraform config merges them, so staged bootstrap output is validated too.

Each file is checked against its JSON schema and against rules the schema cannot
express:

  - duplicate team names / duplicate team memberships
  - a member referencing a team not defined in teams.yaml
  - a protected owner being removed or demoted (--protected-owners)

Files may be absent. An absent members.yaml is treated as an empty member list,
matching how Terraform reads it, so protected owners are enforced even when the
file is deleted outright.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidateOrg(cmd, validateOrgConfigDir, validateOrgProtectedOwners, validateOrgFallbackTeams, validateOrgFallbackMembers)
	},
}

func init() {
	rootCmd.AddCommand(validateOrgCmd)

	validateOrgCmd.Flags().StringVar(&validateOrgConfigDir, "config-dir", "", "Path to the config repository containing the organisation config")
	validateOrgCmd.Flags().StringVar(&validateOrgProtectedOwners, "protected-owners", "", "Comma-separated org logins that must stay owners (never removed or demoted)")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackTeams, "fallback-schema-teams", "", "Fallback teams schema path used when no built-in schema is wanted")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackMembers, "fallback-schema-members", "", "Fallback members schema path used when no built-in schema is wanted")
	_ = validateOrgCmd.MarkFlagRequired("config-dir")
}

func runValidateOrg(cmd *cobra.Command, configDir, protectedOwnersCSV, fallbackTeams, fallbackMembers string) error {
	teamsFiles := existingFiles(
		filepath.Join(configDir, stagedOrgConfigDir, "teams.yaml"),
		filepath.Join(configDir, orgConfigDir, "teams.yaml"),
	)
	membersFiles := existingFiles(
		filepath.Join(configDir, stagedOrgConfigDir, "members.yaml"),
		filepath.Join(configDir, orgConfigDir, "members.yaml"),
	)

	if len(teamsFiles) == 0 && len(membersFiles) == 0 {
		cmd.Println("No organisation teams.yaml or members.yaml found, checking protected owners only")
	}

	var failures []string
	var teamNames []string

	if len(teamsFiles) > 0 {
		schema, err := compileSchema("mem://teams-config.schema.json", fallbackTeams, MarshalTeamsConfigSchema)
		if err != nil {
			return err
		}
		for _, path := range teamsFiles {
			failures = append(failures, validateFile(path, schema)...)

			var cfg github.TeamsConfig
			if err := unmarshalYAMLFile(path, &cfg); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", path, err))
				continue
			}
			for _, e := range cfg.Validate() {
				failures = append(failures, fmt.Sprintf("%s: %v", path, e))
			}
			for _, t := range cfg.Teams {
				teamNames = append(teamNames, t.Name)
			}
		}
	}

	if len(membersFiles) > 0 {
		schema, err := compileSchema("mem://members-config.schema.json", fallbackMembers, MarshalMembersConfigSchema)
		if err != nil {
			return err
		}
		for _, path := range membersFiles {
			failures = append(failures, validateFile(path, schema)...)
		}
	}

	merged, loadErrs := loadMergedMembers(membersFiles)
	failures = append(failures, loadErrs...)

	label := "organisation members config"
	if len(membersFiles) > 0 {
		label = strings.Join(membersFiles, " + ")
	}
	for _, e := range merged.Validate(teamNames, splitCSV(protectedOwnersCSV)) {
		failures = append(failures, fmt.Sprintf("%s: %v", label, e))
	}

	if len(failures) > 0 {
		cmd.PrintErrln("\nOrganisation config validation errors were encountered:")
		for _, f := range failures {
			cmd.PrintErrf("  %s\n", f)
		}
		return fmt.Errorf("organisation config validation failed")
	}

	cmd.Println("ok -- organisation config")
	return nil
}

// loadMergedMembers merges the given members files the same way the Terraform
// config does: later files win per username. With a single file the entries are
// returned as-is, so duplicates within that file are still reported.
func loadMergedMembers(paths []string) (github.MembersConfig, []string) {
	var failures []string
	var configs []github.MembersConfig

	for _, path := range paths {
		var cfg github.MembersConfig
		if err := unmarshalYAMLFile(path, &cfg); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		configs = append(configs, cfg)
	}

	merged := github.MembersConfig{}
	for _, cfg := range configs {
		merged = mergeMembers(merged, cfg)
	}
	return merged, failures
}

func mergeMembers(base, override github.MembersConfig) github.MembersConfig {
	if len(base.Members) == 0 {
		return override
	}
	if len(override.Members) == 0 {
		return base
	}

	overridden := make(map[string]struct{}, len(override.Members))
	for _, m := range override.Members {
		overridden[m.Username] = struct{}{}
	}

	merged := make([]github.Member, 0, len(base.Members)+len(override.Members))
	for _, m := range base.Members {
		if _, replaced := overridden[m.Username]; !replaced {
			merged = append(merged, m)
		}
	}
	merged = append(merged, override.Members...)
	return github.MembersConfig{Members: merged}
}

func existingFiles(paths ...string) []string {
	var out []string
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			out = append(out, path)
		}
	}
	return out
}

func unmarshalYAMLFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	return nil
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
