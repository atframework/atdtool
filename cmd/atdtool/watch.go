package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

const watchDesc = `
This command consists of multiple subcommands which can be used to
observe container resource changes.
`

func newWatchCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Observe container resource changes",
		Long:  watchDesc,
		Args:  require.NoArgs,
	}

	cmd.AddCommand(
		newWatchConfigMapCmd(out),
	)
	return cmd
}

const watchConfigMapDesc = `
Observe configmap resource change.
`

type watchConfigMapOptions struct {
	configPaths      []string
	runCmd           string
	runCmdArgs       []string
	workDir          string
	enableUserSignal bool
	timeout          time.Duration
}

func newWatchConfigMapCmd(out io.Writer) *cobra.Command {
	o := &watchConfigMapOptions{}

	cmd := &cobra.Command{
		Use:   "configmap [PATH]",
		Short: "",
		Long:  watchConfigMapDesc,
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
			o.configPaths = append(o.configPaths, args...)
			return o.run(out)
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.runCmd, "command", "", "run custom command when configuration changes")
	f.StringVarP(&o.workDir, "workdir", "r", "", "specify run command root path")
	f.BoolVar(&o.enableUserSignal, "signal-notify", false, "use user signal to trigger command execution")
	f.StringSliceVar(&o.runCmdArgs, "args", nil, "arguments used by run command, multiple args separated by comma")
	f.DurationVar(&o.timeout, "timeout", 5*time.Minute, "time to wait for command execution")
	return cmd
}

func (o *watchConfigMapOptions) run(_ io.Writer) error {
	signalChan := make(chan os.Signal, 1)
	SetupSignalReload(signalChan)

	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new watcher %v", err)
	}
	defer watcher.Close()

	for _, v := range o.configPaths {
		if err := watcher.Add(v); err != nil {
			return fmt.Errorf("add watch target %v", err)
		}
	}

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if !isValidConfigMapEvent(event) {
					continue
				}

				if err := o.handleEvent(event); err != nil {
					log.Printf("[ERROR] handle event: %v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[ERROR] watch error: %v", err)
			case s := <-signalChan:
				if err := o.handleSignal(s); err != nil {
					log.Printf("[ERROR] handle signal: %v", err)
				}
			}
		}
	}()

	log.Printf("[INFO] configmap watcher has been started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Printf("[WARN] watcher has exited due to %v signal was received!!!", s)
	return nil
}

func (o *watchConfigMapOptions) handleSignal(signal os.Signal) error {
	log.Printf("[INFO] received %v", signal)
	return o.runCustomCmd()
}

func (o *watchConfigMapOptions) handleEvent(event fsnotify.Event) error {
	log.Printf("[INFO] received event %v", event)
	return o.runCustomCmd()
}

func (o *watchConfigMapOptions) runCustomCmd() error {
	if o.runCmd == "" {
		return nil
	}

	ctx, cancle := context.WithTimeout(context.Background(), o.timeout)
	defer cancle()
	cmd := exec.CommandContext(ctx, o.runCmd, o.runCmdArgs...)
	if o.workDir != "" {
		cmd.Dir = o.workDir
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	sigs := make(chan os.Signal, 1)
	SetupSignalChild(cmd, sigs)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command %v", err)
	}

	log.Printf("[INFO] command execution output: %s", out.String())
	return nil
}

func isValidConfigMapEvent(event fsnotify.Event) bool {
	if event.Op&fsnotify.Create != fsnotify.Create {
		return false
	}
	if filepath.Base(event.Name) != "..data" {
		return false
	}
	return true
}
