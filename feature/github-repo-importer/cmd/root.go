package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "importer",
	Short:         "A CLI tool to fetch GitHub repository details, branch protection rules & rulesets",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			_, _ = fmt.Fprintf(os.Stdout, "::error::%s\n", escapeAnnotation(err.Error()))
		}
		os.Exit(1)
	}
}

// escapeAnnotation escapes a message for use in a GitHub Actions workflow command.
func escapeAnnotation(s string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return r.Replace(s)
}
