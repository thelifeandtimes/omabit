package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

func main() {
	if os.Getpid() != 1 || os.Geteuid() != 0 {
		os.Exit(80)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		panic(err)
	}
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil || !strings.Contains(string(cmdline), "omabit.disposable-guest=smoke-v1") {
		panic("missing disposable guest marker")
	}
	if err := syscall.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		panic(err)
	}
	vendor, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err != nil || strings.TrimSpace(string(vendor)) != "QEMU" {
		panic("not a QEMU guest")
	}
	if _, err := os.Stat("/home"); !os.IsNotExist(err) {
		panic("unexpected home mount")
	}
	fmt.Println("OMABIT_VM_READY disposable QEMU guest, PID 1, no home or network device")
	syscall.Sync()
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		panic(err)
	}
}
