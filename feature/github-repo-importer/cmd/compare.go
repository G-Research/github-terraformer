package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/gr-oss-devops/github-repo-importer/pkg/compare"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare [dir1] [dir2]",
	Short: "Compare command compares two directories and generates a diff",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dirA := args[0]
		dirB := args[1]

		result, err := compare.CompareDirectories(dirA, dirB)
		if err != nil {
			return fmt.Errorf("Error comparing directories: %w\n", err)
		}

		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)
}
