//go:build !linux

package main

import "os/exec"

func protectClipboardChild(*exec.Cmd) {}
