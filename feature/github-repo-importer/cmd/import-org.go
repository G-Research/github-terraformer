package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gr-oss-devops/github-repo-importer/pkg/github"
)

var importOrgOrg string

var importOrgCmd = &cobra.Command{
	Use:   "import-org",
	Short: "Import an organisation's teams and members into organisation/teams.yaml and organisation/members.yaml",
	PreRun: func(cmd *cobra.Command, args []string) {
		github.InitializeClients()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		teams, members, err := github.ImportOrg(importOrgOrg)
		if err != nil {
			return fmt.Errorf("failed to import organisation: %w", err)
		}

		if err := github.WriteOrgConfig(importOrgOrg, teams, members); err != nil {
			return fmt.Errorf("failed to write org config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Imported %d teams and %d members for %s\n", len(teams.Teams), len(members.Members), importOrgOrg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importOrgCmd)
	importOrgCmd.Flags().StringVarP(&importOrgOrg, "org", "o", "", "GitHub organisation to import")
	_ = importOrgCmd.MarkFlagRequired("org")
}
