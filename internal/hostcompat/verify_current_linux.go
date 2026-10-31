//go:build linux

package hostcompat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type boundedOutput struct {
	buf    bytes.Buffer
	limit  int64
	cancel context.CancelFunc
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if int64(b.buf.Len())+int64(len(p)) > b.limit {
		b.cancel()
		return 0, errEvidence
	}
	return b.buf.Write(p)
}

func nativeRunner(root rootFiles) commandRunner {
	return func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		switch tool {
		case "/usr/bin/gpg", "/usr/bin/gpgv", "/usr/bin/bsdtar", "/usr/bin/dpkg-deb", "/usr/lib/apt/apt-helper":
		default:
			return nil, errEvidence
		}
		_, info, err := resolve(root.FS(), strings.TrimPrefix(tool, "/"))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return nil, errEvidence
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		out := &boundedOutput{limit: limit, cancel: cancel}
		cmd := exec.CommandContext(ctx, tool, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return os.ErrProcessDone
			}
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				return err
			}
			return os.ErrProcessDone
		}
		cmd.Env = []string{"PATH=/usr/bin:/usr/sbin:/bin:/sbin", "HOME=/root", "LANG=C", "LC_ALL=C"}
		cmd.Dir = "/"
		cmd.Stdout, cmd.Stderr, cmd.WaitDelay = out, io.Discard, time.Second
		if err := cmd.Run(); err != nil {
			// Nonzero gpgv status can mean extra signatures whose public keys
			// were intentionally excluded by Signed-By. Preserve bounded status
			// for per-signature evaluation; do not relax any other command.
			statusOnly := false
			for i, arg := range args {
				if arg == "--status-fd" && i+1 < len(args) && args[i+1] == "1" {
					statusOnly = true
				}
				if arg == "--output" {
					statusOnly = false
					break
				}
			}
			var exit *exec.ExitError
			if tool == "/usr/bin/gpgv" && statusOnly && ctx.Err() == nil && errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 2) {
				return out.buf.Bytes(), nativeSignatureExit{exit.ExitCode()}
			}
			return nil, errors.New("native package evidence validation failed")
		}
		return out.buf.Bytes(), nil
	}
}

func nativeStreamRunner(root *os.Root) streamRunner {
	return func(ctx context.Context, tool string, args []string, limit int64, consume func(io.Reader) error) error {
		switch tool {
		case "/usr/lib/apt/apt-helper", "/usr/bin/bsdtar", "/usr/bin/dpkg-deb":
		default:
			return errEvidence
		}
		_, info, err := resolve(root.FS(), strings.TrimPrefix(tool, "/"))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return errEvidence
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, tool, args...)
		cmd.Env = []string{"PATH=/usr/bin:/usr/sbin:/bin:/sbin", "HOME=/root", "LANG=C", "LC_ALL=C"}
		cmd.Dir = "/"
		cmd.Stderr = io.Discard
		cmd.WaitDelay = time.Second
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return os.ErrProcessDone
			}
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				return err
			}
			return os.ErrProcessDone
		}
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return errEvidence
		}
		if cmd.Start() != nil {
			return errEvidence
		}
		count := &byteCounter{Reader: io.LimitReader(pipe, limit+1)}
		err = consume(count)
		if err == nil {
			_, err = io.Copy(io.Discard, count)
		}
		if err != nil || count.n > limit {
			cancel()
		}
		waitErr := cmd.Wait()
		if err != nil || waitErr != nil || count.n > limit || ctx.Err() != nil {
			return errEvidence
		}
		return nil
	}
}

// VerifyCurrent explicitly refreshes authentication using already-cached native
// package artifacts. It requires local root authorization, has a two-minute total
// deadline, never fetches/installs packages/imports keys, and writes only local
// root-owned verification evidence. Partial failures withdraw authorization while
// retaining historical signed proof for an offline retry.
func VerifyCurrent(ctx context.Context) (Report, error) {
	return VerifyCurrentWithOptions(ctx, VerificationOptions{})
}

// VerifyCurrentWithOptions adds explicit, local-root-only download of missing
// exact installed builds into private staging. The default remains offline.
func VerifyCurrentWithOptions(ctx context.Context, options VerificationOptions) (Report, error) {
	if ctx == nil {
		return Report{}, errors.New("local package verification requires a context")
	}
	if ctx.Err() != nil {
		return Report{}, errors.New("local package verification was canceled")
	}
	if os.Geteuid() != 0 {
		return InspectCurrent(), errors.New("local root authorization is required to refresh package authentication")
	}
	deadline := 2 * time.Minute
	if options.FetchMissing {
		deadline = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	root, err := os.OpenRoot("/")
	if err != nil {
		return Report{}, errEvidence
	}
	defer root.Close()
	if prepareVerificationDirectory(root) != nil {
		return Inspect(root.FS(), runtime.GOOS, runtime.GOARCH), errors.New("trusted verification directory is unavailable")
	}
	unlock, err := lockRefresh(ctx, root)
	if err != nil {
		return Report{}, errors.New("local package verification is busy or canceled")
	}
	defer unlock()
	fsys, work := withWork(currentEvidenceFS(root.FS()), verificationLimits, &fileDigestCache{})
	snapshot := inspect(fsys, runtime.GOOS, runtime.GOARCH)
	old, _ := loadReceipts(root.FS(), snapshot.report.Distribution.Provider)
	v := verifier{fsys: fsys, run: boundCommands(nativeRunner(root), work), stream: boundStreams(nativeStreamRunner(root), work), store: cacheStore(budgetRoot{root, fsys})}
	var activeFetcher *nativeFetcher
	if options.FetchMissing {
		fetcher, cleanup, err := newNativeFetcher(root, &v)
		if err != nil {
			return snapshot.report, errors.New("private native download staging is unavailable")
		}
		defer cleanup()
		activeFetcher = fetcher
		v.fetch = fetcher.fetch
	}
	set, failed, refreshErr := v.refreshPackages(ctx, snapshot, old)
	if refreshErr != nil {
		// Publish revocation even when current trust is unavailable or changes
		// during refresh. Preserve old signed proof for a later local retry.
		set.Packages = old.Packages
		for i := range set.Packages {
			set.Packages[i].Authorized = false
		}
	}
	if err := writeReceipts(root, set); err != nil {
		return snapshot.report, errors.New("could not atomically store trusted package authentication receipts")
	}
	pruneVerificationCache(root, set)
	report := Inspect(currentEvidenceFS(root.FS()), runtime.GOOS, runtime.GOARCH)
	if work.exhausted {
		report.Checks = append(report.Checks, Check{"packages.verification-budget", "unknown", "Explicit verification exceeded its aggregate I/O, command-output or record budget; unresolved packages were not authenticated."})
	}
	if refreshErr != nil || failed || report.Provenance != "verified" {
		if activeFetcher != nil && activeFetcher.lastError == errAPTSandboxUnavailable {
			return report, errAPTSandboxUnavailable
		}
		return report, packageVerificationError(report)
	}
	return report, nil
}

func writeReceipts(root *os.Root, set receiptSet) error {
	b, err := json.Marshal(set)
	if err != nil || len(b) > 1<<20 {
		return errEvidence
	}
	return atomicRootFile(root, receiptPath, b, 0644)
}
