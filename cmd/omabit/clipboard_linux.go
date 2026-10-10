//go:build linux

package main

import (
	"os/exec"
	"syscall"
)

func protectClipboardChild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
