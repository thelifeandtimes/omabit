//go:build linux && amd64

// PID 1 for the private initramfs. Never run this program on the host.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"golang.org/x/sys/unix"
)

func event(fields map[string]any) {
	b, _ := json.Marshal(fields)
	fmt.Printf("POWERCUT %s\n", b)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func command(path string, args ...string) {
	c := exec.Command(path, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	must(c.Run())
}

func main() {
	if os.Getpid() != 1 {
		fmt.Fprintln(os.Stderr, "powercut init requires private VM PID 1")
		os.Exit(2)
	}
	defer func() {
		if r := recover(); r != nil {
			event(map[string]any{"event": "failure", "error": fmt.Sprint(r)})
			// No sync on failure: preserve the failure state for inspection.
			_ = unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
		}
	}()
	marker, err := os.ReadFile("/powercut-guest")
	must(err)
	if string(marker) != "isolated-rc12-filesystem-guest-v1\n" {
		panic("wrong initramfs")
	}
	must(unix.Mount("proc", "/proc", "proc", 0, ""))
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""))
	must(unix.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""))
	cmdline, err := os.ReadFile("/proc/cmdline")
	must(err)
	options := map[string]string{}
	for _, part := range strings.Fields(string(cmdline)) {
		if k, v, ok := strings.Cut(part, "="); ok {
			options[k] = v
		}
	}
	phase, fs, suite, cut := options["powercut.phase"], options["powercut.fs"], options["powercut.suite"], options["powercut.case"]
	if (phase != "seed" && phase != "crash" && phase != "verify") || (fs != "ext4" && fs != "btrfs") || (suite != "storage" && suite != "update") {
		panic("invalid guest options")
	}
	modules, err := os.ReadFile("/powercut-modules")
	must(err)
	for _, name := range strings.Fields(string(modules)) {
		f, err := os.Open(name)
		must(err)
		err = unix.FinitModule(int(f.Fd()), "", 0)
		f.Close()
		must(err)
	}
	for _, device := range []string{"/dev/vda", "/dev/vdb"} {
		for n := 0; ; n++ {
			if _, err := os.Stat(device); err == nil {
				break
			}
			if n == 100 {
				panic("virtio disk unavailable")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if phase == "seed" {
			if fs == "ext4" {
				command("/usr/bin/mkfs.ext4", "-F", "-q", "-E", "lazy_itable_init=0,lazy_journal_init=0", device)
			} else {
				command("/usr/bin/mkfs.btrfs", "-f", "-q", device)
			}
		}
	}
	must(unix.Mount("/dev/vda", "/source", fs, 0, ""))
	must(unix.Mount("/dev/vdb", "/destination", fs, 0, ""))
	version, _ := os.ReadFile("/proc/version")
	mounts, _ := os.ReadFile("/proc/mounts")
	event(map[string]any{"event": "boot", "phase": phase, "filesystem": fs, "kernel": strings.TrimSpace(string(version)), "mounts": string(mounts), "memory_mib": 768})
	command("/usr/bin/mkfs.btrfs", "--version")
	command("/usr/bin/mkfs.ext4", "-V")
	c := exec.Command("/fixture.test", "-test.run=^TestRC12PowercutGuest$", "-test.v", "-test.timeout=90s")
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	c.Env = []string{"PATH=/usr/bin", "HOME=/", "GOMAXPROCS=1", "GODEBUG=asyncpreemptoff=1",
		"OMABIT_POWERCUT_PHASE=" + phase, "OMABIT_POWERCUT_SUITE=" + suite, "OMABIT_POWERCUT_CASE=" + cut}
	if phase == "crash" {
		must(trace(c, cut))
		panic("fixture finished without requested power cut")
	}
	must(c.Run())
	// Seed establishes a known durable baseline. Verification is a separate boot
	// with ordinary filesystem journal replay, never a host mount or fsck repair.
	unix.Sync()
	must(unix.Unmount("/source", 0))
	must(unix.Unmount("/destination", 0))
	event(map[string]any{"event": "success", "phase": phase})
	must(unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF))
}

// Linux PTRACE_GET_SYSCALL_INFO distinguishes entry/exit even for new Go threads
// and signal restarts. No production code is rewritten or injected into.
type syscallInfo struct {
	Op     uint8
	Pad    [3]byte
	Arch   uint32
	IP, SP uint64
	Data   [8]uint64
}

type operation struct {
	Name string `json:"syscall"`
	Path string `json:"path"`
	To   string `json:"to,omitempty"`
}

func traceString(pid int, address uint64) string {
	var result []byte
	for len(result) < 4096 {
		var chunk [8]byte
		_, err := unix.PtracePeekData(pid, uintptr(address)+uintptr(len(result)), chunk[:])
		must(err)
		for _, b := range chunk {
			if b == 0 {
				return string(result)
			}
			result = append(result, b)
		}
	}
	panic("unbounded syscall pathname")
}

func fdPath(pid int, fd uint64) string {
	name := fmt.Sprintf("/proc/%d/fd/%d", pid, fd)
	if int32(fd) == unix.AT_FDCWD {
		name = fmt.Sprintf("/proc/%d/cwd", pid)
	}
	p, err := os.Readlink(name)
	must(err)
	return p
}

func argumentPath(pid int, fd, address uint64) string {
	p := traceString(pid, address)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(fdPath(pid, fd), p)
}

func identify(pid int, info syscallInfo) operation {
	nr, a := info.Data[0], info.Data[1:]
	switch nr {
	case unix.SYS_FSYNC, unix.SYS_FDATASYNC:
		return operation{Name: "fsync", Path: fdPath(pid, a[0])}
	case unix.SYS_UNLINKAT:
		return operation{Name: "unlink", Path: argumentPath(pid, a[0], a[1])}
	case unix.SYS_RENAMEAT, unix.SYS_RENAMEAT2:
		return operation{Name: "rename", Path: argumentPath(pid, a[0], a[1]), To: argumentPath(pid, a[2], a[3])}
	case unix.SYS_RENAME:
		return operation{Name: "rename", Path: traceString(pid, a[0]), To: traceString(pid, a[1])}
	}
	return operation{}
}

func matches(cut string, op operation, side string, counts map[string]int) bool {
	const dest = "/destination/home/piers/fakeships/fixture/pier"
	const held = "/source/held/pier"
	const journal = "/destination/etc/urbit-runner/management-activation.json"
	const executable = "/destination" + access.ShellPath
	base := ""
	switch {
	case op.Name == "fsync" && op.Path == dest+"/.urb/chk/image.bin":
		base = "copy-file-" + side + "-fsync"
	case op.Name == "fsync" && op.Path == dest:
		base = "destination-leaf-" + side + "-fsync"
	case op.Name == "fsync" && op.Path == filepath.Dir(dest):
		base = "destination-mount-" + side + "-fsync"
	case op.Name == "fsync" && op.Path == "/destination/home/piers/fakeships":
		base = "destination-group-" + side + "-fsync"
	case op.Name == "fsync" && op.Path == "/destination/home/piers":
		base = "destination-root-" + side + "-fsync"
	case op.Name == "fsync" && op.Path == "/destination/home":
		base = "destination-" + side + "-fsync"
	case op.Name == "unlink" && op.Path == held+"/.urb/log/nested/07.bin":
		base = "source-" + side + "-unlink"
	case op.Name == "fsync" && op.Path == held+"/.urb/log/nested":
		base = "source-parent-" + side + "-fsync"
	case op.Name == "unlink" && op.Path == held:
		base = "source-root-" + side + "-unlink"
	case op.Name == "fsync" && op.Path == "/source/held" && side == "after":
		base = "source-removed"
	case op.Name == "fsync" && strings.HasPrefix(op.Path, filepath.Dir(journal)+"/.omabit-install-"):
		base = "journal-" + side + "-file-fsync"
	case op.Name == "fsync" && strings.HasPrefix(op.Path, filepath.Dir(executable)+"/.omabit-install-"):
		base = "executable-" + side + "-file-fsync"
	case op.Name == "rename" && op.To == journal:
		base = "journal-" + side + "-rename"
	case op.Name == "rename" && op.To == executable:
		base = "executable-" + side + "-rename"
	case op.Name == "fsync" && op.Path == filepath.Dir(journal):
		base = "journal-" + side + "-dir-fsync"
		if cut == "starting-after-dir-fsync" && side == "after" && counts[base] == 1 {
			return true
		}
	case op.Name == "fsync" && op.Path == filepath.Dir(executable):
		base = "executable-" + side + "-dir-fsync"
	}
	counts[base]++
	return base != "" && base == cut && counts[base] == 1
}

func trace(c *exec.Cmd, cut string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	c.SysProcAttr = &syscall.SysProcAttr{Ptrace: true}
	if err := c.Start(); err != nil {
		return err
	}
	mainPID := c.Process.Pid
	var status unix.WaitStatus
	_, err := unix.Wait4(mainPID, &status, 0, nil)
	must(err)
	must(unix.PtraceSetOptions(mainPID, unix.PTRACE_O_TRACESYSGOOD|unix.PTRACE_O_TRACECLONE|unix.PTRACE_O_EXITKILL))
	must(unix.PtraceSyscall(mainPID, 0))
	pending := map[int]operation{}
	counts := map[string]int{}
	sequence := 0
	for {
		pid, err := unix.Wait4(-1, &status, unix.WALL, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		must(err)
		if status.Exited() || status.Signaled() {
			delete(pending, pid)
			if pid == mainPID {
				return fmt.Errorf("fixture exited before cut: %v", status)
			}
			continue
		}
		sig := 0
		if status.StopSignal() == unix.SIGTRAP|0x80 {
			var info syscallInfo
			_, _, errno := unix.Syscall6(unix.SYS_PTRACE, 0x420e, uintptr(pid), unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info)), 0, 0)
			if errno != 0 {
				return errno
			}
			op, side := operation{}, ""
			if info.Op == 1 {
				op, side = identify(pid, info), "before"
				pending[pid] = op
			} else if info.Op == 2 {
				op, side = pending[pid], "after"
				delete(pending, pid)
				if int64(info.Data[0]) != 0 {
					op = operation{}
				}
			}
			if _, err := os.Stat("/powercut-armed"); err == nil && op.Name != "" {
				sequence++
				e := map[string]any{"event": "syscall", "sequence": sequence, "tid": pid, "side": side, "operation": op}
				if side == "after" {
					e["return"] = int64(info.Data[0])
				}
				if matches(cut, op, side, counts) {
					e["event"], e["case"] = "boundary", cut
					e["tracee_stopped"] = true
					event(e)
					// The mutating thread remains in ptrace-stop. The host's only
					// next action is SIGKILL of QEMU, never a guest sync/shutdown.
					for {
						time.Sleep(time.Hour)
					}
				}
				event(e)
			}
		} else if status.StopSignal() != unix.SIGTRAP && status.StopSignal() != unix.SIGSTOP {
			sig = int(status.StopSignal())
		}
		must(unix.PtraceSyscall(pid, sig))
	}
}
