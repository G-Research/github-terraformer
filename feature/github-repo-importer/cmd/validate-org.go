package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

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
file is deleted outright.

Unlike repository config, the schema is never taken from the config repository:
both the schema and the protected-owner list are deployment config, so a pull
request cannot weaken the rules it is validated against.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidateOrg(cmd, validateOrgConfigDir, validateOrgProtectedOwners, validateOrgFallbackTeams, validateOrgFallbackMembers)
	},
}

func init() {
	rootCmd.AddCommand(validateOrgCmd)

	validateOrgCmd.Flags().StringVar(&validateOrgConfigDir, "config-dir", "", "Path to the config repository containing the organisation config")
	validateOrgCmd.Flags().StringVar(&validateOrgProtectedOwners, "protected-owners", "", "Comma-separated org logins that must stay owners (never removed or demoted)")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackTeams, "fallback-schema-teams", "", "Path to the teams schema to validate against; defaults to the schema built into this binary")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackMembers, "fallback-schema-members", "", "Path to the members schema to validate against; defaults to the schema built into this binary")
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

	protectedOwners := splitCSV(protectedOwnersCSV)
	if len(protectedOwners) > 0 {
		cmd.Printf("Enforcing %d protected owner(s): %s\n", len(protectedOwners), strings.Join(protectedOwners, ", "))
	} else {
		cmd.PrintErrln("WARNING: no protected owners configured, owner removal and demotion are not enforced")
	}

	if len(teamsFiles) == 0 && len(membersFiles) == 0 {
		cmd.Println("No organisation teams.yaml or members.yaml found, validating as an empty organisation config")
	}

	var failures []string

	teamNames, teamsOK := validateTeamsFiles(cmd, teamsFiles, fallbackTeams, &failures)
	validateMembersFiles(cmd, membersFiles, fallbackMembers, teamNames, teamsOK, protectedOwners, &failures)

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

// validateTeamsFiles returns the team names declared across the given files and
// whether every file loaded.
func validateTeamsFiles(cmd *cobra.Command, paths []string, fallbackSchema string, failures *[]string) ([]string, bool) {
	if len(paths) == 0 {
		return nil, true
	}

	schema, err := compileSchema("mem://teams-config.schema.json", fallbackSchema, MarshalTeamsConfigSchema)
	if err != nil {
		*failures = append(*failures, fmt.Sprintf("teams schema: %v", err))
		return nil, false
	}

	var teamNames []string
	loadedAll := true
	for _, path := range paths {
		data, instance, loadErr := loadYAMLDocument(path)
		if loadErr != nil {
			*failures = append(*failures, fmt.Sprintf("%s: %v", path, loadErr))
			loadedAll = false
			continue
		}
		*failures = append(*failures, validateInstance(path, instance, schema)...)

		var cfg github.TeamsConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			*failures = append(*failures, fmt.Sprintf("%s: does not match the teams config shape: %v", path, err))
			loadedAll = false
			continue
		}
		for _, e := range cfg.Validate() {
			*failures = append(*failures, fmt.Sprintf("%s: %v", path, e))
		}
		for _, t := range cfg.Teams {
			teamNames = append(teamNames, t.Name)
		}
	}
	return teamNames, loadedAll
}

func validateMembersFiles(cmd *cobra.Command, paths []string, fallbackSchema string, teamNames []string, teamsOK bool, protectedOwners []string, failures *[]string) {
	var schema *jsonschema.Schema
	if len(paths) > 0 {
		compiled, err := compileSchema("mem://members-config.schema.json", fallbackSchema, MarshalMembersConfigSchema)
		if err != nil {
			*failures = append(*failures, fmt.Sprintf("members schema: %v", err))
		} else {
			schema = compiled
		}
	}

	merged := github.MembersConfig{}
	loadedAll := true
	for _, path := range paths {
		data, instance, loadErr := loadYAMLDocument(path)
		if loadErr != nil {
			*failures = append(*failures, fmt.Sprintf("%s: %v", path, loadErr))
			loadedAll = false
			continue
		}
		if schema != nil {
			*failures = append(*failures, validateInstance(path, instance, schema)...)
		}

		var cfg github.MembersConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			*failures = append(*failures, fmt.Sprintf("%s: does not match the members config shape: %v", path, err))
			loadedAll = false
			continue
		}

		if teamsOK {
			for _, e := range cfg.ValidateEntries(teamNames) {
				*failures = append(*failures, fmt.Sprintf("%s: %v", path, e))
			}
		}

		merged = mergeMembers(merged, cfg)
	}

	if !teamsOK {
		cmd.PrintErrln("WARNING: skipping member team-reference checks until the teams config loads")
	}
	if !loadedAll {
		return
	}

	label := "organisation members config"
	if len(paths) > 0 {
		label = strings.Join(paths, " + ")
	}
	for _, e := range merged.ValidateProtectedOwners(protectedOwners) {
		*failures = append(*failures, fmt.Sprintf("%s: %v", label, e))
	}
}

// mergeMembers merges two member sets the way the Terraform config does: override
// wins per username.
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

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
