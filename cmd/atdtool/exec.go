package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

const execDesc = `Used to run custom command`

type execOptions struct {
	runCmd     string
	runCmdArgs []string
	workDir    string
	timeout    time.Duration
}

func newExecCmd(out io.Writer) *cobra.Command {
	o := &execOptions{}

	cmd := &cobra.Command{
		Use:   "exec [CMD]",
		Short: "Run custom command",
		Long:  execDesc,
		Args:  require.MinimumNArgs(1),
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
			o.runCmd = args[0]
			return o.run(out)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.workDir, "workdir", "r", "", "specify run command root path")
	f.StringSliceVar(&o.runCmdArgs, "args", nil, "arguments used by run command, multiple args separated by comma")
	f.DurationVar(&o.timeout, "timeout", 5*time.Minute, "time to wait for command execution")
	return cmd
}

func (o *execOptions) run(out io.Writer) error {
	if o.runCmd == "" {
		return nil
	}

	ctx, cancle := context.WithTimeout(context.Background(), o.timeout)
	defer cancle()
	cmd := exec.CommandContext(ctx, o.runCmd, o.runCmdArgs...)
	if o.workDir != "" {
		cmd.Dir = o.workDir
	}

	cmd.Stdout = out
	cmd.Stderr = out

	sigs := make(chan os.Signal, 1)
	SetupSignalChild(cmd, sigs)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command %v", err)
	}
	return nil
}
