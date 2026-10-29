//go:build linux && amd64

// Private VM PID 1. This program must never be invoked as a host service.
package main

import (
	"fmt"
	"os"
	"path/filepath"
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
			fmt.Println("OMABIT_GUEST_INIT_FAILURE", value)
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
	if !strings.Contains(string(cmdline), "omabit.disposable-guest=offline-import-v1") || strings.TrimSpace(string(vendor)) != "QEMU" {
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
	must(unix.Mount("/dev/vda", "/newroot", "ext4", 0, ""))
	marker, err := os.ReadFile("/newroot/opt/omabit-lab/guest-marker")
	must(err)
	if string(marker) != "offline-import-guest-v1\n" {
		panic("wrong fresh root disk")
	}
	// The host builds a regular disk file without privileges. Normalize ownership
	// only inside this new guest, before systemd or package installation can run.
	must(filepath.Walk("/newroot", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err = os.Lchown(path, 0, 0); err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return os.Chmod(path, info.Mode())
		}
		return nil
	}))
	for _, name := range []string{"proc", "sys", "dev"} {
		must(os.MkdirAll("/newroot/"+name, 0755))
		must(unix.Mount("/"+name, "/newroot/"+name, "", unix.MS_MOVE, ""))
	}
	must(os.Chdir("/newroot"))
	must(unix.Mount(".", "/", "", unix.MS_MOVE, ""))
	must(unix.Chroot("."))
	must(os.Chdir("/"))
	fmt.Println("OMABIT_GUEST_INIT booting real systemd on the fresh private disk")
	must(unix.Exec("/usr/lib/systemd/systemd", []string{"/usr/lib/systemd/systemd"}, []string{"PATH=/usr/bin:/bin", "HOME=/root", "LANG=C.UTF-8"}))
}
