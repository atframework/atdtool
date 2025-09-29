package main

import (
	"bytes"
	"io"
	"os"

	"github.com/atframework/atdtool/cli/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	toolName = "atdtool"

	globalUsage = `Configuration management tool

Common actions for atdtool:

- atdtool template:      Render custom chart templates
`
)

// ToolName returns the tool name.
func ToolName() string {
	return toolName
}

func addValueOptionsFlags(f *pflag.FlagSet, v *values.Options) {
	f.StringSliceVarP(&v.Paths, "values", "p", []string{}, "set values path on the command line (can specify multiple paths with commas:path1,path2)")
	f.StringArrayVarP(&v.Values, "set", "s", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
}

func newRootCmd(out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "atdtool",
		Short:        "The configuration management for service.",
		Long:         globalUsage,
		SilenceUsage: true,
	}

	flags := cmd.PersistentFlags()
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Parse(args)

	// Add subcommands
	cmd.AddCommand(
		newVersionCmd(out),
		newTemplateCmd(out),
		newMergeValuesCmd(out),
		newGUIDCmd(out),
		newWatchCmd(out),
		newExecCmd(out),
	)

	return cmd, nil
}

func main() {
	var out bytes.Buffer
	cmd, err := newRootCmd(&out, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		out.WriteTo(os.Stderr)
		os.Exit(1)
	}
	out.WriteTo(os.Stdout)
}
