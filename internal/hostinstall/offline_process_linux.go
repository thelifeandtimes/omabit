package hostinstall

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func validateHistoricalExecutable(binary []byte) error {
	f, err := elf.NewFile(bytes.NewReader(binary))
	if err != nil {
		return errors.New("historical runner must be an ELF executable")
	}
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Machine != elf.EM_X86_64 || f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
		return errors.New("historical runner requires Linux amd64 ELF provenance")
	}
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			return errors.New("offline schema probe requires the static release runner, without a dynamic loader")
		}
	}
	info, err := buildinfo.Read(bytes.NewReader(binary))
	if err != nil || info.Path != "git.thechristophgroup.org/thelifeandtimes/omabit/backend/cmd/urbit-runner" || info.Main.Path != "git.thechristophgroup.org/thelifeandtimes/omabit" {
		return errors.New("installed artifact is not a recognized Go urbit-runner executable")
	}
	settings := map[string]string{}
	for _, v := range info.Settings {
		settings[v.Key] = v.Value
	}
	if settings["GOOS"] != "linux" || settings["GOARCH"] != "amd64" {
		return errors.New("installed runner build target is unknown")
	}
	return nil
}

// Only the previously installed, root-owned artifact is probed. The child execs
// after dropping to the existing owner inside a fresh chroot containing only a
// root-owned 0555 runner. There are no writable files/directories, /proc, home,
// sockets or devices in that root, and a fresh network namespace has no host
// connectivity. Namespace/credential isolation failure is a hard refusal.
func (i *installer) historicalSchema(ctx context.Context, binary []byte, p ownerhost.Profile) (int, error) {
	if i.offlineArtifactSchema != nil {
		return i.offlineArtifactSchema(ctx, binary, p)
	}
	if err := validateHistoricalExecutable(binary); err != nil {
		return 0, err
	}
	if err := i.parents(ctx, generationsPath+"/probe", true); err != nil {
		return 0, err
	}
	dir, err := os.MkdirTemp(i.path(generationsPath), ".schema-probe-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	logical := filepath.Join(generationsPath, filepath.Base(dir), "runner")
	if err := i.installFile(ctx, logical, binary, 0555, false); err != nil {
		return 0, err
	}
	if err := os.Chmod(dir, 0555); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := historicalSchemaCommand(ctx, dir, p)
	data, err := runOfflineChild(cmd)
	if err != nil {
		return 0, fmt.Errorf("isolated unprivileged historical schema probe failed: %w", err)
	}
	schema, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || schema != 19 {
		return 0, errors.New("offline compatibility path requires the original installed schema19 runner; unknown historical schema refused")
	}
	return schema, nil
}

func historicalSchemaCommand(ctx context.Context, root string, p ownerhost.Profile) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "/runner", "state-schema")
	cmd.Dir, cmd.Env = "/", []string{"PATH=/", "HOME=/", "LANG=C"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Chroot: root, Credential: &syscall.Credential{Uid: p.UID, Gid: p.GID, Groups: []uint32{}}, Cloneflags: syscall.CLONE_NEWNET | syscall.CLONE_NEWNS, Setpgid: true, Pdeathsig: syscall.SIGKILL}
	return cmd
}

// Only the currently authorized installer executable runs this helper, after
// setuid/setgid and with the ownership flock inherited as descriptor 3. SQL WAL
// files are consequently created by the pier owner, never by the root installer.
func (i *installer) offlineState(ctx context.Context, lock *managed.OfflineStateLock, p ownerhost.Profile, req managed.OfflineUpdateRequest) (managed.OfflineUpdateState, error) {
	if i.offlineStateCall != nil {
		return i.offlineStateCall(ctx, lock, p, req)
	}
	var result managed.OfflineUpdateState
	if err := lock.Verify(); err != nil {
		return result, err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "management-offline-state")
	cmd.Dir, cmd.Env = "/", []string{"PATH=/usr/bin:/bin", "HOME=" + p.Home, "LANG=C"}
	cmd.Stdin, cmd.ExtraFiles = bytes.NewReader(data), []*os.File{lock.File()}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: p.UID, Gid: p.GID, Groups: []uint32{}}, Setpgid: true, Pdeathsig: syscall.SIGKILL}
	out, err := runOfflineChild(cmd)
	if err != nil {
		return result, err
	}
	d := json.NewDecoder(bytes.NewReader(out))
	d.DisallowUnknownFields()
	if d.Decode(&result) != nil || d.Decode(new(any)) != io.EOF {
		return result, errors.New("invalid offline state helper response")
	}
	return result, lock.Verify()
}

func runOfflineChild(cmd *exec.Cmd) ([]byte, error) {
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	var stdout, stderr boundedOutput
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil || stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("offline helper failed: %v (%s)", err, strconv.Quote(string(stderr.buffer.Bytes()[:min(stderr.buffer.Len(), 2048)])))
	}
	return stdout.buffer.Bytes(), nil
}
