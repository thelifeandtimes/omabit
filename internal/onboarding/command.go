package onboarding

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh"
)

func desktopKey(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := readPrivate(path, 16384)
	if errors.Is(err, os.ErrNotExist) {
		if _, err := os.Lstat(path + ".pub"); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("Desktop public key exists without its private key; restore the key before setup.")
		}
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		block, err := ssh.MarshalPrivateKey(key, "omabit-local")
		if err != nil {
			return nil, err
		}
		data = pem.EncodeToMemory(block)
		if err := writePrivate(path, data); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil || signer.PublicKey().Type() != ssh.KeyAlgoED25519 {
		return nil, errors.New("The desktop setup key is unavailable or modified; it was preserved.")
	}
	return ssh.MarshalAuthorizedKey(signer.PublicKey()), nil
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
	return installHostExpected(ctx, runner, bundle, public, "", emit)
}

func installHostExpected(ctx context.Context, runner, bundle, public, expected string, emit func(Event)) (hostinstall.HostResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	program := "/usr/bin/pkexec"
	if isatty.IsTerminal(os.Stdin.Fd()) {
		program = "/usr/bin/sudo"
	}
	owner, err := user.Current()
	if err != nil {
		return hostinstall.HostResult{}, err
	}
	args := []string{runner, "install-host", "--owner", owner.Username, "--bundle", bundle, "--public-key", public, "--acknowledge-host-changes"}
	if expected != "" {
		args = append(args, "--expected-runner-id", expected)
	} else {
		args = append(args, "--install-prerequisites")
	}
	cmd := exec.CommandContext(ctx, program, args...)
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
		return hostinstall.HostResult{}, errors.New("Could not open OS authorization. Check that the desktop authentication agent is running, then retry.")
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
			message.Message = map[string]string{"packages": "Downloading and installing missing native packages", "package-verify": "Verifying native packages and host prerequisites", "engine": "Preparing dedicated rootless Docker", "access": "Configuring private desktop access", "services": "Installing background services", "verify": "Waiting for management readiness", "host": "Host setup complete"}[message.Step]
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
		return result, errors.New("Host setup did not finish. Authorization may have been cancelled; review the current step and retry when ready.")
	}
	return result, nil
}
