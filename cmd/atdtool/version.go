package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

var (
	toolVersion string
)

// ToolName returns the tool version.
func ToolVersion() string {
	return toolVersion
}

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the atdtool version information",
		Long:  "Print the atdtool version information",
		Args:  require.ExactArgs(0),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// Allow file completion when completing the argument for the name
				// which could be a path
				return nil, cobra.ShellCompDirectiveDefault
			}
			// No more completions, so disable file completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(out, "%s %s %s/%s\n", toolName, toolVersion, runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
	return cmd
}
