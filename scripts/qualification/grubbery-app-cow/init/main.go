//go:build linux && amd64

// Private derived-fixture VM PID 1; original backing is read-only. This program must never be invoked as a host service.
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
	marker, err := os.ReadFile("/newroot/opt/omabit-issues-results/result.json")
	must(err)
	var prior struct {
		Source string `json:"source_commit"`
		Passed bool   `json:"passed"`
		Inputs struct {
			Kind string `json:"kind"`
		} `json:"inputs"`
	}
	must(json.Unmarshal(marker, &prior))
	if prior.Source != "5eeae9a2c9c15c764aaa6534b6a9fd6d745f6bb4" || prior.Passed || prior.Inputs.Kind != "issues18-product-grubbery-v1" {
		panic("wrong fixed disposable predecessor")
	}
	if _, err = os.Lstat("/newroot/opt/omabit-issues-nec-results"); !os.IsNotExist(err) {
		panic("never resume a derived capsule")
	}
	must(unix.Mount("/dev/vdb", "/newroot/opt/omabit-release-input", "ext4", unix.MS_RDONLY, "noload"))
	// Suppress startup resume BEFORE systemd/user services can run. Only this new
	// COW overlay gets the override; the fixed original root is mounted read-only
	// by QEMU. Guest authorizes the new manager after checking old evidence.
	must(os.MkdirAll("/newroot/etc/systemd/user/urbit-runner.service.d", 0755))
	override := `[Unit]
ConditionPathExists=/run/omabit-nec-runner-ready
[Service]
ExecStart=
ExecStart=/usr/bin/env -i "HOME=%h" PATH=/usr/bin:/bin LANG=C "XDG_RUNTIME_DIR=%t" /opt/omabit-release-input/candidate/urbit-runner serve-development --acknowledge-development
`
	must(os.WriteFile("/newroot/etc/systemd/user/urbit-runner.service.d/99-issues-fresh-nec.conf", []byte(override), 0644))
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
	// The inherited existing symlink is preserved and must target this exact unit.
	target, err := os.Readlink("/newroot/etc/systemd/system/multi-user.target.wants/omabit-release-upgrade.service")
	must(err)
	if target != "../omabit-release-upgrade.service" {
		panic("foreign predecessor service")
	}
	for _, name := range []string{"proc", "sys", "dev"} {
		must(os.MkdirAll("/newroot/"+name, 0755))
		must(unix.Mount("/"+name, "/newroot/"+name, "", unix.MS_MOVE, ""))
	}
	must(os.Chdir("/newroot"))
	must(unix.Mount(".", "/", "", unix.MS_MOVE, ""))
	must(unix.Chroot("."))
	must(os.Chdir("/"))
	fmt.Println("OMABIT_GUEST_INIT booting systemd on a fresh COW fixture; predecessor never writable")
	must(unix.Exec("/usr/lib/systemd/systemd", []string{"/usr/lib/systemd/systemd"}, []string{"PATH=/usr/bin:/bin", "HOME=/root", "LANG=C.UTF-8"}))
}
