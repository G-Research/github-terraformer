package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "importer",
	Short: "A CLI tool to fetch GitHub repository details, branch protection rules & rulesets",
	// Don't dump command usage/help on runtime errors — they bury the real
	// message. We print errors ourselves in Execute below.
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		// Surface a clean annotation in the GitHub Actions UI/run summary.
		// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			fmt.Printf("::error::%s\n", escapeAnnotation(err.Error()))
		}
		os.Exit(1)
	}
}

// escapeAnnotation escapes a message for use in a GitHub Actions workflow command.
func escapeAnnotation(s string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return r.Replace(s)
}
