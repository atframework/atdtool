package main

import (
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"sigs.k8s.io/yaml"

	"github.com/atframework/atdtool/cli/values"
	"github.com/atframework/atdtool/internal/pkg/util"
)

const mergeValuesDesc = `
To generate values in a chart, use either the '--values' flag and pass file
path or use the '--set' flag and pass configuration from the command line.

You can specify the multiple replace paths with '--values'/'-p' flag.
Multiple paths are separated by commas. The priority will be given to the last 
(right-most) path specified.

You can specify the '--set'/'-s' flag multiple times. The priority will be given to the
last (right-most) set specified.
`

type mergeValuesOptions struct {
	chartPath string
	outPath   string
	valOpts   values.Options
}

func newMergeValuesCmd(out io.Writer) *cobra.Command {
	o := &mergeValuesOptions{}

	cmd := &cobra.Command{
		Use:   "merge-values [CHART]",
		Short: "Generate values file in a chart",
		Long:  mergeValuesDesc,
		Args:  require.ExactArgs(1),
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
			o.chartPath = args[0]
			if err := o.run(out); err != nil {
				return err
			}
			return nil
		},
	}

	if out != nil {
		cmd.SetOut(out)
	}

	f := cmd.Flags()
	addValueOptionsFlags(f, &o.valOpts)
	f.StringVarP(&o.outPath, "output", "o", "", "specify values file save path")
	return cmd
}

func (o *mergeValuesOptions) run(_ io.Writer) (err error) {
	var (
		valuePaths []string
		optVals    map[string]any
		vals       map[string]any
	)

	valuePaths, err = o.valOpts.MergePaths()
	if err != nil {
		return
	}

	optVals, err = o.valOpts.MergeValues()
	if err != nil {
		return
	}

	vals, err = util.MergeChartValues(o.chartPath, valuePaths, optVals, nil)
	if err != nil {
		return
	}

	var out []byte
	out, err = yaml.Marshal(vals)
	if err != nil {
		return
	}

	var filename string
	if o.outPath != "" {
		filename = o.outPath
		if filepath.Ext(o.outPath) == "" || filepath.Ext(o.outPath) == "." {
			filename = filepath.Join(filename, "values.yaml")
		}
	} else {
		filename = filepath.Join(o.chartPath, "values.yaml")
	}
	err = util.WriteFile(out, filename)
	return
}
