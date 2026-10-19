//go:build linux

package hostcompat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

const maxFetchBytes int64 = 2 << 30
const maxFetchFiles = 2048

func newNativeFetcher(root *os.Root, v *verifier) (*nativeFetcher, func(), error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, errFetch
	}
	stage := strings.TrimPrefix(VerificationDirectory, "/") + "/.fetch-" + hex.EncodeToString(nonce[:])
	if err := root.Mkdir(stage, 0700); err != nil {
		return nil, nil, errFetch
	}
	identity, err := root.Lstat(stage)
	if err != nil {
		return nil, nil, errFetch
	}
	cleanup := func() {
		current, err := root.Lstat(stage)
		if err == nil && os.SameFile(identity, current) {
			root.RemoveAll(stage)
		}
	}
	mkdir := func(name string) error {
		if name != stage && !strings.HasPrefix(name, stage+"/") {
			return errFetch
		}
		if err := root.Mkdir(name, 0700); err != nil && !os.IsExist(err) {
			return errFetch
		}
		resolved, info, err := resolve(root.FS(), name)
		if err != nil || resolved != name || !info.IsDir() {
			return errFetch
		}
		return nil
	}
	write := func(name string, b []byte) error {
		if !strings.HasPrefix(name, stage+"/") || len(b) > 1<<20 {
			return errFetch
		}
		return atomicRootFile(root, name, b, 0600)
	}
	for _, name := range []string{stage + "/home", stage + "/tmp"} {
		if err := mkdir(name); err != nil {
			cleanup()
			return nil, nil, errFetch
		}
	}
	policy := &downloadPolicy{stage: stage}
	f := &nativeFetcher{v: v, stage: stage, mkdir: mkdir, write: write, run: nativeFetchRunner(root, stage, policy)}
	f.prepareAPT = func(base string) error { return prepareAPTSandbox(root, policy, v.fsys, base) }
	f.sealAPT = func(ctx context.Context, base string, owner nativePackage, indexes bool) (string, error) {
		return sealAPTDownloads(ctx, root, policy, v, base, owner, indexes)
	}
	return f, cleanup, nil
}

// Native downloads run in a new private directory with no inherited package
// manager hooks/config overrides. prlimit enforces a hard per-file ceiling; a
// bounded tree monitor enforces aggregate staging limits and rejects aliases.
// No program output, URLs, credentials or raw stderr are returned as diagnostics.
func nativeFetchRunner(root *os.Root, stage string, policy *downloadPolicy) fetchRunner {
	return func(ctx context.Context, request fetchCommand) ([]byte, error) {
		switch request.Tool {
		case "/usr/bin/apt-get", "/usr/bin/pacman-conf", "/usr/bin/curl":
		default:
			return nil, errFetch
		}
		if !strings.HasPrefix(request.Directory, "/"+stage) || (request.Directory != "/"+stage && !strings.HasPrefix(request.Directory, "/"+stage+"/")) || path.Clean(request.Directory) != request.Directory {
			return nil, errFetch
		}
		for _, tool := range []string{request.Tool, "/usr/bin/prlimit"} {
			_, info, err := resolve(root.FS(), strings.TrimPrefix(tool, "/"))
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
				return nil, errFetch
			}
		}
		if policy.checkPath(root, strings.TrimPrefix(request.Directory, "/")) != nil {
			return nil, errFetch
		}
		dir, err := root.Lstat(strings.TrimPrefix(request.Directory, "/"))
		if err != nil || !dir.IsDir() {
			return nil, errFetch
		}
		env := []string{"PATH=/usr/bin:/usr/sbin:/bin:/sbin", "HOME=/" + stage + "/home", "TMPDIR=/" + stage + "/tmp", "LANG=C", "LC_ALL=C"}
		if request.Tool == "/usr/bin/apt-get" {
			if !policy.aptReady {
				return nil, errAPTSandboxUnavailable
			}
			if !strings.HasPrefix(request.APTConfig, "/"+stage+"/") {
				return nil, errFetch
			}
			if _, err := readTrusted(root.FS(), strings.TrimPrefix(request.APTConfig, "/"), 1<<20); err != nil {
				return nil, errFetch
			}
			env = append(env, "APT_CONFIG="+request.APTConfig)
			env = []string{"PATH=/usr/bin:/usr/sbin:/bin:/sbin", "HOME=/" + stage + "/apt/home", "TMPDIR=/" + stage + "/apt/tmp", "LANG=C", "LC_ALL=C", "APT_CONFIG=" + request.APTConfig}
		} else if request.APTConfig != "" {
			return nil, errFetch
		}
		limit := request.MaxFile
		if limit <= 0 {
			limit = 1 << 20
		}
		if limit > maxArtifact {
			return nil, errFetch
		}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		out := &boundedOutput{limit: 4 << 20, cancel: cancel}
		args := append([]string{"--fsize=" + fmt.Sprint(limit) + ":" + fmt.Sprint(limit), "--", request.Tool}, request.Args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/prlimit", args...)
		cmd.Env, cmd.Dir = env, request.Directory
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if request.Tool == "/usr/bin/apt-get" {
			cmd.SysProcAttr.Credential = aptCredential(policy)
		}
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return os.ErrProcessDone
			}
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				return err
			}
			return os.ErrProcessDone
		}
		cmd.Stdout, cmd.Stderr, cmd.WaitDelay = out, io.Discard, time.Second
		if stageWithinBoundsPolicy(root, stage, policy) != nil {
			return nil, errFetch
		}
		if err := cmd.Start(); err != nil {
			return nil, errFetch
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case err := <-done:
				if err != nil || ctx.Err() != nil || stageWithinBoundsPolicy(root, stage, policy) != nil {
					return nil, errFetch
				}
				return out.buf.Bytes(), nil
			case <-ticker.C:
				if stageWithinBoundsPolicy(root, stage, policy) != nil {
					cancel()
					<-done
					return nil, errFetch
				}
			case <-ctx.Done():
				cancel()
				<-done
				return nil, errFetch
			}
		}
	}
}

func stageWithinBounds(root rootFiles, stage string) error {
	return stageWithinBoundsPolicy(root, stage, &downloadPolicy{stage: stage})
}

func stageWithinBoundsPolicy(root rootFiles, stage string, policy *downloadPolicy) error {
	var bytes int64
	count := 0
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if depth > 8 {
			return errFetch
		}
		entries, err := acquisitionEntries(root, policy, dir)
		if err != nil {
			if _, statErr := root.Lstat(dir); os.IsNotExist(statErr) {
				return nil
			}
			return errFetch
		}
		for _, entry := range entries {
			count++
			if count > maxFetchFiles {
				return errFetch
			}
			name := dir + "/" + entry.Name()
			info, err := root.Lstat(name)
			// Native acquisition atomically renames/removes partial files. A
			// disappeared entry is not a staging escape or an over-budget file.
			if os.IsNotExist(err) {
				continue
			}
			if err != nil || !policy.allows(name, info) {
				return errFetch
			}
			if info.IsDir() {
				if err := walk(name, depth+1); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxArtifact {
				return errFetch
			}
			bytes += info.Size()
			if bytes > maxFetchBytes {
				return errFetch
			}
		}
		return nil
	}
	return walk(stage, 0)
}

func aptCredential(policy *downloadPolicy) *syscall.Credential {
	return &syscall.Credential{Uid: policy.aptUID, Gid: policy.aptGID, Groups: []uint32{}}
}
