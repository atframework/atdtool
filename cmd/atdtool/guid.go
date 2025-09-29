package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"

	"github.com/atframework/atdtool/pkg/snowflake"
)

const gUIDDesc = `
This command consists of multiple subcommands which can be used to
manager guid.
`

func newGUIDCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guid",
		Short: "Global unique id generator",
		Long:  gUIDDesc,
		Args:  require.NoArgs,
	}

	cmd.AddCommand(
		newGenGUIDCmd(out),
	)
	return cmd
}

const genGUIDDesc = `
Generate global unique id.
`

type genGUIDOptions struct {
	algorithm string
}

func newGenGUIDCmd(out io.Writer) *cobra.Command {
	o := &genGUIDOptions{}

	cmd := &cobra.Command{
		Use:   "gen",
		Short: "",
		Long:  genGUIDDesc,
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
			return o.run(out)
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.algorithm, "algorithm", "", "specify generate gloabl unique id algorithm")
	return cmd
}

func (o *genGUIDOptions) run(out io.Writer) error {
	s := snowflake.NewSnowFlake(nil)
	val, err := s.NextVal()
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%d\n", val)
	return err
}
