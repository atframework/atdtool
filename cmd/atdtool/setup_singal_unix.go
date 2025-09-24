//go:build linux || darwin
// +build linux darwin

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func SetupSignalChild(cmd *exec.Cmd, sigs chan<- os.Signal) {
	signal.Notify(sigs, syscall.SIGCHLD)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func SetupSignalReload(sigs chan<- os.Signal) {
	signal.Notify(sigs, syscall.SIGUSR1)
}
