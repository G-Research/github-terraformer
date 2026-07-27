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

var (
	validateOrgConfigDir       string
	validateOrgProtectedOwners string
	validateOrgFallbackTeams   string
	validateOrgFallbackMembers string
)

var validateOrgCmd = &cobra.Command{
	Use:   "validate-org",
	Short: "Validate organisation/teams.yaml and organisation/members.yaml before Terraform runs",
	Long: `ValidateOrg validates organisation/teams.yaml and organisation/members.yaml in
the config directory, failing fast before Terraform runs. Each file is checked
against its JSON schema and against cross-file rules the schema cannot express:

  - duplicate team names / duplicate team memberships
  - a member referencing a team not defined in teams.yaml
  - a protected owner being removed or demoted (--protected-owners)

Either file may be absent; validation runs on whichever exist.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidateOrg(cmd, validateOrgConfigDir, validateOrgProtectedOwners, validateOrgFallbackTeams, validateOrgFallbackMembers)
	},
}

func init() {
	rootCmd.AddCommand(validateOrgCmd)

	validateOrgCmd.Flags().StringVar(&validateOrgConfigDir, "config-dir", "", "Path to the config repository containing organisation/*.yaml")
	validateOrgCmd.Flags().StringVar(&validateOrgProtectedOwners, "protected-owners", "", "Comma-separated org logins that must stay owners (never removed or demoted)")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackTeams, "fallback-schema-teams", "", "Fallback teams schema path used when no built-in schema is wanted")
	validateOrgCmd.Flags().StringVar(&validateOrgFallbackMembers, "fallback-schema-members", "", "Fallback members schema path used when no built-in schema is wanted")
	_ = validateOrgCmd.MarkFlagRequired("config-dir")
}

func runValidateOrg(cmd *cobra.Command, configDir, protectedOwnersCSV, fallbackTeams, fallbackMembers string) error {
	teamsPath := filepath.Join(configDir, "organisation", "teams.yaml")
	membersPath := filepath.Join(configDir, "organisation", "members.yaml")

	teamsExists := fileExists(teamsPath)
	membersExists := fileExists(membersPath)

	if !teamsExists && !membersExists {
		cmd.Println("No organisation/teams.yaml or organisation/members.yaml found, skipping validation")
		return nil
	}

	var failures []string
	var teamNames []string

	if teamsExists {
		schema, err := compileSchema("mem://teams-config.schema.json", fallbackTeams, MarshalTeamsConfigSchema)
		if err != nil {
			return err
		}
		failures = append(failures, validateFile(teamsPath, schema)...)

		var teamsCfg github.TeamsConfig
		if err := unmarshalYAMLFile(teamsPath, &teamsCfg); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", teamsPath, err))
		} else {
			for _, e := range teamsCfg.Validate() {
				failures = append(failures, fmt.Sprintf("%s: %v", teamsPath, e))
			}
			for _, t := range teamsCfg.Teams {
				teamNames = append(teamNames, t.Name)
			}
		}
	}

	if membersExists {
		schema, err := compileSchema("mem://members-config.schema.json", fallbackMembers, MarshalMembersConfigSchema)
		if err != nil {
			return err
		}
		failures = append(failures, validateFile(membersPath, schema)...)

		var membersCfg github.MembersConfig
		if err := unmarshalYAMLFile(membersPath, &membersCfg); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", membersPath, err))
		} else {
			for _, e := range membersCfg.Validate(teamNames, splitCSV(protectedOwnersCSV)) {
				failures = append(failures, fmt.Sprintf("%s: %v", membersPath, e))
			}
		}
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
