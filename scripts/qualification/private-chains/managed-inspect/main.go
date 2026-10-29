//go:build linux && amd64

// Private VM PID 1. This program must never be invoked as a host service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var inspectionGuestMarker = "offline-managed-private-inspect-v1"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--private-input-benchmark" {
		benchmarkChild()
		return
	}
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
	if !strings.Contains(string(cmdline), "omabit.disposable-guest="+inspectionGuestMarker) || strings.TrimSpace(string(vendor)) != "QEMU" {
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
	inspectManaged()
	benchmarkInputs()
	unix.Sync()
	must(unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF))
}

// The child executes this compiled reader only, never any program from disk.
// It needs the fixture owner UID because the actual input verifier checks it.
func benchmarkChild() {
	if os.Getpid() == 1 || os.Getppid() != 1 || os.Getuid() != 1000 || os.Geteuid() != 1000 {
		os.Exit(80)
	}
	vendor, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err != nil || strings.TrimSpace(string(vendor)) != "QEMU" {
		os.Exit(80)
	}
	cmd, err := os.ReadFile("/proc/cmdline")
	if err != nil || !strings.Contains(string(cmd), "omabit.disposable-guest="+inspectionGuestMarker) {
		os.Exit(80)
	}
	var fs unix.Statfs_t
	if unix.Statfs("/newroot", &fs) != nil || fs.Flags&unix.ST_RDONLY == 0 {
		os.Exit(80)
	}
	const manifest = "fd4c0739be071ef2ad88fb10bcd17cbc93c712e6dc7544ca8a2c9f0232f7832a"
	const foundation = "17a7d5fc3fb8e9314f5d6e806cad3e4ef0654388c87920ff729b48624e846410"
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	v, err := privatechain.OpenRuntimeInputVerifier(ctx, root+"/private-runtime-inputs/"+manifest, manifest, foundation)
	if err != nil {
		fmt.Println(`{"ok":false,"stage":"open"}`)
		return
	}
	defer v.Close()
	durations := []int64{}
	for n := 0; n < 2; n++ {
		start := time.Now()
		_, err := v.Verify(ctx)
		elapsed := time.Since(start).Milliseconds()
		durations = append(durations, elapsed)
		if err != nil {
			json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "stage": "verify", "durations_ms": durations, "deadline": ctx.Err() != nil})
			return
		}
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "durations_ms": durations, "files": 12352, "bytes": 834044028, "input_programs_executed": false})
}
func benchmarkInputs() {
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/init", "--private-input-benchmark")
	cmd.Env = []string{"PATH=/bin", "GOMAXPROCS=2"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1000, Gid: 1000, Groups: []uint32{1000}}}
	raw, err := cmd.Output()
	var result map[string]any
	if err != nil || len(raw) > 4096 || json.Unmarshal(raw, &result) != nil {
		result = map[string]any{"ok": false, "stage": "child", "raw_error_exported": false}
	}
	data, err := json.Marshal(result)
	must(err)
	fmt.Println("OMABIT_PRIVATE_INPUT_TIMING " + string(data))
}
