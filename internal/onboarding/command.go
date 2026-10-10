package onboarding

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"github.com/mattn/go-isatty"
)

func desktopKey(ctx context.Context, path string) ([]byte, error) {
	if _, err := readPrivate(path, 16384); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Lstat(path + ".pub"); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("Desktop public key exists without its private key; restore the key before setup.")
		}
		stage, err := os.MkdirTemp(filepath.Dir(path), "key-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(stage)
		key := filepath.Join(stage, "identity")
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(c, "/usr/bin/ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "omabit-local", "-f", key)
		if err := cmd.Run(); err != nil {
			return nil, errors.New("Could not create the private desktop access key.")
		}
		if err := os.Rename(key, path); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "/usr/bin/ssh-keygen", "-y", "-P", "", "-f", path)
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil || output.Len() > 8192 {
		return nil, errors.New("The desktop setup key is unavailable or modified; it was preserved.")
	}
	return output.Bytes(), nil
}

type boundedStderr struct{ data bytes.Buffer }

func (b *boundedStderr) Write(p []byte) (int, error) {
	n := len(p)
	if b.data.Len() < 8192 {
		left := 8192 - b.data.Len()
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.data.Write(p)
	}
	return n, nil
}

func installHost(ctx context.Context, runner, bundle, public string, emit func(Event)) (hostinstall.HostResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	program := "/usr/bin/pkexec"
	if isatty.IsTerminal(os.Stdin.Fd()) {
		program = "/usr/bin/sudo"
	}
	cmd := exec.CommandContext(ctx, program, runner, "install-host", "--bundle", bundle, "--public-key", public, "--acknowledge-host-changes")
	cmd.WaitDelay = 2 * time.Second
	if program == "/usr/bin/sudo" {
		cmd.Stdin = os.Stdin
	}
	stderr := &boundedStderr{}
	if program == "/usr/bin/sudo" {
		cmd.Stderr = io.MultiWriter(os.Stderr, stderr)
	} else {
		cmd.Stderr = stderr
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return hostinstall.HostResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return hostinstall.HostResult{}, errors.New("Could not open OS authorization. Run omabit --format text setup from a terminal.")
	}
	var result hostinstall.HostResult
	var failure string
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 64<<10)
	for scanner.Scan() {
		var message struct {
			Step, Status, Message string
			Host                  *hostinstall.HostResult `json:"host"`
		}
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			failure = "Invalid host setup response."
			cancel()
			break
		}
		if message.Status == "failed" {
			failure = message.Message
		}
		if message.Host != nil {
			result = *message.Host
		}
		if message.Message == "" {
			message.Message = map[string]string{"engine": "Preparing dedicated rootless Docker", "access": "Configuring private desktop access", "services": "Installing background services", "verify": "Waiting for management readiness", "host": "Host setup complete"}[message.Step]
		}
		if emit != nil {
			emit(Event{Step: message.Step, Status: message.Status, Message: message.Message})
		}
	}
	err = errors.Join(scanner.Err(), cmd.Wait())
	if err != nil || !result.Ready {
		if failure != "" {
			return result, errors.New(failure)
		}
		if strings.Contains(stderr.data.String(), "dismissed") || strings.Contains(stderr.data.String(), "authorized") {
			return result, errors.New("Host setup was not authorized. Your existing installation was preserved; try again when ready.")
		}
		return result, errors.New("Host setup did not finish. Authorization may have been cancelled; inspect the current step or run omabit --format text setup in a terminal before retrying.")
	}
	return result, nil
}
