//go:build linux && amd64

// Private VM PID 1. This program must never be invoked as a host service.
package main

import (
	"encoding/json"
	"fmt"
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
	if !strings.Contains(string(cmdline), "omabit.disposable-guest=release-upgrade-v1") || strings.TrimSpace(string(vendor)) != "QEMU" {
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
	marker, err := os.ReadFile("/newroot/opt/omabit-review/fresh-onboarding.json")
	must(err)
	var m struct {
		State string `json:"state"`
		Ships int    `json:"ships_enrolled"`
	}
	must(json.Unmarshal(marker, &m))
	if m.State != "not-onboarded" || m.Ships != 0 {
		panic("not the pristine onboarding base")
	}
	if _, err = os.Stat("/newroot/opt/omabit-release-input"); !os.IsNotExist(err) {
		panic("release guest already provisioned")
	}
	must(os.Mkdir("/newroot/opt/omabit-release-input", 0755))
	must(unix.Mount("/dev/vdb", "/newroot/opt/omabit-release-input", "ext4", unix.MS_RDONLY, "noload"))
	// Only the newly created overlay receives this service. No networking or
	// host device/service is mounted into the guest.
	service := `[Unit]
Description=Disposable Omabit release upgrade qualification
After=systemd-user-sessions.service
[Service]
Type=oneshot
ExecStart=/usr/bin/python3 -B /opt/omabit-release-input/guest.py
StandardOutput=journal+console
StandardError=journal+console
TimeoutStartSec=2400
[Install]
WantedBy=multi-user.target
`
	must(os.WriteFile("/newroot/etc/systemd/system/omabit-release-upgrade.service", []byte(service), 0644))
	must(os.Symlink("../omabit-release-upgrade.service", "/newroot/etc/systemd/system/multi-user.target.wants/omabit-release-upgrade.service"))
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
