//go:build windows
// +build windows

package main

import (
	"os"
	"os/exec"
)

func SetupSignalChild(_cmd *exec.Cmd, _sigs chan<- os.Signal) {
}

func SetupSignalReload(_sigs chan<- os.Signal) {
}
