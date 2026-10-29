//go:build linux && amd64

// Private VM PID 1. This program must never be invoked as a host service.
package main

import (
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if os.Getpid() != 1 || os.Geteuid() != 0 {
		os.Exit(80)
	}
	defer func() {
		if value := recover(); value != nil {
			emit(map[string]any{"schema": 1, "ok": false, "reader_failure": readerFailureProjection(value), "raw_state_exported": false})
			unix.Sync()
			_ = unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
		}
	}()
	must(unix.Mount("proc", "/proc", "proc", 0, ""))
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""))
	must(unix.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""))
	cmdline, err := os.ReadFile("/proc/cmdline")
	must(err)
	vendor, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	must(err)
	if !strings.Contains(string(cmdline), "omabit.disposable-guest=offline-private-inspect-v1") || strings.TrimSpace(string(vendor)) != "QEMU" {
		panic("wrong disposable guest")
	}
	modules, err := os.ReadFile("/guest-modules")
	must(err)
	for _, path := range strings.Fields(string(modules)) {
		f, err := os.Open(path)
		must(err)
		err = unix.FinitModule(int(f.Fd()), "", 0)
		f.Close()
		must(err)
	}
	for n := 0; ; n++ {
		if _, err := os.Stat("/dev/vda"); err == nil {
			break
		}
		if n == 100 {
			panic("guest disk unavailable")
		}
		time.Sleep(20 * time.Millisecond)
	}
	must(unix.Mount("/dev/vda", "/newroot", "ext4", unix.MS_RDONLY, "noload"))
	marker, err := os.ReadFile("/newroot/opt/omabit-lab/guest-marker")
	must(err)
	if string(marker) != "offline-import-guest-v1\n" {
		panic("wrong fresh root disk")
	}
	resume := resumeRequired(string(cmdline))
	peers := gatewayPeersRequired(string(cmdline), resume)
	publication := publicationRequired(string(cmdline), peers)
	initial := initialPublicationRequired(string(cmdline), resume || peers || publication)
	if isolationRequired(string(cmdline), resume || peers || publication || initial) {
		inspectIsolation()
	} else {
		inspectResults(resume, peers, publication, initial)
	}
	unix.Sync()
	must(unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF))
}
