//go:build linux && amd64

// Read-only result extractor, PID1 in a new no-NIC VM. Never boots the inspected
// filesystem's systemd, services, containers or ships and never replays its log.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func inspectionGuest(command, vendor string) bool {
	return slices.Contains(strings.Fields(command), "omabit.disposable-guest=issues-readonly-inspection-v1") && strings.TrimSpace(vendor) == "QEMU"
}

func main() {
	if os.Getpid() != 1 || os.Geteuid() != 0 {
		os.Exit(80)
	}
	defer func() {
		if v := recover(); v != nil {
			b, _ := json.Marshal(map[string]any{"passed": false, "error": fmt.Sprint(v)})
			fmt.Println("OMABIT_ISSUES_INSPECTION " + string(b))
		}
		_ = unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}()
	must(unix.Mount("proc", "/proc", "proc", 0, ""))
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""))
	must(unix.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""))
	command, err := os.ReadFile("/proc/cmdline")
	must(err)
	vendor, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	must(err)
	if !inspectionGuest(string(command), string(vendor)) {
		panic("wrong inspection VM")
	}
	modules, err := os.ReadFile("/guest-modules")
	must(err)
	for _, path := range strings.Fields(string(modules)) {
		f, e := os.Open(path)
		must(e)
		e = unix.FinitModule(int(f.Fd()), "", 0)
		f.Close()
		must(e)
	}
	for n := 0; ; n++ {
		if _, err := os.Stat("/dev/vda"); err == nil {
			break
		}
		if n == 100 {
			panic("inspection block unavailable")
		}
		time.Sleep(20 * time.Millisecond)
	}
	must(unix.Mount("/dev/vda", "/newroot", "ext4", unix.MS_RDONLY|unix.MS_NODEV|unix.MS_NOSUID|unix.MS_NOEXEC, "noload"))
	read := func(path string) []byte {
		f, e := os.Open("/newroot/" + path)
		must(e)
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
		must(e)
		if len(b) > 4*1024*1024 {
			panic("bounded result exceeded")
		}
		return b
	}
	var scenario struct {
		Inputs struct {
			Kind string `json:"kind"`
		} `json:"inputs"`
	}
	result := read("opt/omabit-issues-results/result.json")
	must(json.Unmarshal(result, &scenario))
	if scenario.Inputs.Kind != "issues16-21-development-v1" {
		panic("wrong inspected scenario")
	}
	files := map[string][]byte{}
	for _, name := range []string{"commands.jsonl", "result.json"} {
		files[name] = read("opt/omabit-issues-results/" + name)
	}
	// Explicit private archival of this fresh fixture's original writer bytes,
	// before retiring an obsolete full disk. Never interpreted or repaired here.
	if slices.Contains(strings.Fields(string(command)), "omabit.inspect-original-store=1") {
		for _, name := range []string{"runner.sqlite", "runner.sqlite-wal", "runner.sqlite-shm"} {
			path := "home/review/.local/share/urbit-runner/managed/" + name
			if _, err := os.Stat("/newroot/" + path); os.IsNotExist(err) && name != "runner.sqlite" {
				continue
			}
			files[name] = read(path)
		}
	}
	b, err := json.Marshal(map[string]any{"passed": true, "read_only": true, "journal_replayed": false, "services_started": false, "files": files})
	must(err)
	fmt.Println("OMABIT_ISSUES_INSPECTION " + string(b))
}
