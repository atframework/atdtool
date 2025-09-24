package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/atframework/atdtool/internal/pkg/logarchive"
	_ "github.com/atframework/atdtool/internal/pkg/logarchive/modules/cos"
	_ "github.com/atframework/atdtool/internal/pkg/logarchive/modules/filearchive"
)

const (
	ExitCodeSuccess = iota
	ExitCodeFailedStartup
	ExitCodeForceQuit
	ExitCodeFailedQuit
)

var (
	toolName    = "log-archive"
	toolVersion string
	configFile  string

	globalUsage = `Used to collect log from multiple inputs to the specified output
Common actions for log-archive:

- log-archive start:      Starts the log-archive process and blocks indefinitely
- log-archive version:    Prints the version
`
)

// exactArgs returns an error if there are not exactly n args.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf(
				"%q requires %d arguments\n\nUsage:  %s",
				cmd.CommandPath(),
				n,
				cmd.UseLine(),
			)
		}
		return nil
	}
}

// ToolName returns the tool name.
func ToolName() string {
	return toolName
}

func newRootCmd(out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "log-archive",
		Short:        "Tool used for log archive.",
		Long:         globalUsage,
		SilenceUsage: true,
	}
	flags := cmd.PersistentFlags()

	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Parse(args)

	// Add subcommands
	cmd.AddCommand(
		newVersionCmd(out),
		newStartCmd(out),
	)

	return cmd, nil
}

// ToolName returns the tool version.
func ToolVersion() string {
	return toolVersion
}

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version of log-archive",
		Long:  "Prints the version of log-archive",
		Args:  exactArgs(0),
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

func newStartCmd(_ io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the log-archive process in the background",
		Long:  "Starts the log-archive process in the background",
		Args:  exactArgs(0),
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
			return startProcess()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&configFile, "config", "c", "", "Configuration file")
	return cmd
}

func startProcess() error {
	// trap signal
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

		for sig := range sigchan {
			switch sig {
			case syscall.SIGQUIT:
				os.Exit(ExitCodeForceQuit)
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				if err := logarchive.Stop(); err != nil {
					os.Exit(ExitCodeFailedQuit)
				}
				os.Exit(ExitCodeSuccess)
			}
		}
	}()

	config, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read log-archive config file: %v", err)
	}

	success, exit := make(chan struct{}), make(chan error)
	// start server
	go func() {
		if err := logarchive.Start(config); err != nil {
			exit <- err
		} else {
			close(success)
		}
	}()

	select {
	case <-success:
		fmt.Printf("Successfully started log-archive\n")
	case err := <-exit:
		return err
	}

	// block
	select {}
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
