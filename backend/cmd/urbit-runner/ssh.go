package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

// OpenSSH invokes the account's restricted login shell with -c and the forced
// command from authorized_keys. The client-provided original command is fixed.
func runSSHShell(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	u, err := user.Current()
	if err != nil || u.Username != "urbit-runner" || len(args) != 2 || args[0] != "-c" || os.Getenv("SSH_ORIGINAL_COMMAND") != client.SSHCommand {
		fmt.Fprintln(stderr, "restricted runner SSH access only")
		return 1
	}
	parts := strings.Fields(args[1])
	if len(parts) != 3 || parts[0] != client.SSHCommand {
		return 1
	}
	grant, err := access.Load(access.DefaultKeys, parts[1])
	if err != nil || grant.Role != parts[2] {
		return 1
	}
	socket := "/run/user/" + strconv.Itoa(os.Geteuid()) + "/urbit-runner/runner.sock"
	if err := sshBridge(ctx, os.Stdin, stdout, socket, access.DefaultKeys, grant.Fingerprint); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runOwnerSSH(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	p, err := ownerhost.Current()
	if err != nil || p == nil || len(args) != 2 || os.Getenv("SSH_ORIGINAL_COMMAND") != client.SSHCommand {
		fmt.Fprintln(stderr, "restricted owner RPC access only")
		return 1
	}
	g, err := access.Load(access.DefaultKeys, args[0])
	if err != nil || g.Role != args[1] {
		return 1
	}
	socket := "/run/user/" + strconv.Itoa(os.Geteuid()) + "/urbit-runner/runner.sock"
	if err = sshBridge(ctx, os.Stdin, stdout, socket, access.DefaultKeys, g.Fingerprint); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runSSH(ctx context.Context, command string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var public, role, socket, keys, fingerprint string
	var output string
	var port uint
	if command == "ssh-authorized-key" || command == "prepare-ssh" {
		flags.StringVar(&public, "public-key", "", "Ed25519 public key file")
		flags.StringVar(&role, "role", "", "operator or observer")
		if command == "prepare-ssh" {
			flags.StringVar(&output, "output", "", "new absolute directory for inert review artifacts")
			flags.UintVar(&port, "port", 22222, "loopback-only development SSH port")
		}
	} else {
		flags.StringVar(&socket, "socket", "", "local private runner socket")
		flags.StringVar(&keys, "authorized-keys", "", "reviewed restricted authorized_keys file")
		flags.StringVar(&fingerprint, "fingerprint", "", "sshd forced-command grant identity")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		return 2
	}
	if command == "ssh-authorized-key" || command == "prepare-ssh" {
		f, err := os.Open(public)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, 8193))
		if err != nil {
			return 1
		}
		if command == "prepare-ssh" {
			u, err := user.Current()
			if err == nil {
				err = access.PrepareOwner(output, data, role, port, u.Username)
			}
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			fmt.Fprintf(stdout, "Prepared inert owner SSH review files in %s\nListen address: 127.0.0.1:%d\nForced RPC executable: %s\nHost installation requires administrative authorization; the login shell is preserved.\n", output, port, access.ShellPath)
			return 0
		}
		line, err := access.OwnerAuthorizedKey(data, role)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, line)
		return 0
	}
	// This explicit helper is for reviewed forced commands and isolated fixtures.
	// It cannot grant access beyond the invoking Unix account's socket permissions.
	if !filepath.IsAbs(socket) || !filepath.IsAbs(keys) || os.Getenv("SSH_ORIGINAL_COMMAND") != client.SSHCommand {
		fmt.Fprintln(stderr, "require fixed SSH command and absolute local paths")
		return 2
	}
	if _, err := access.Load(keys, fingerprint); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := sshBridge(ctx, os.Stdin, stdout, socket, keys, fingerprint); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// Inherited SSH stdio may be blocking descriptors outside Go's poller. Merely
// closing os.Stdin/os.Stdout does not interrupt their pending syscalls. Give the
// bridge session-owned, pollable duplicates so cancellation releases both waits.
func sshBridge(ctx context.Context, input *os.File, output io.Writer, socket, keys, fingerprint string) error {
	in, err := sshStream(input)
	if err != nil {
		return err
	}
	defer in.Close()
	if f, ok := output.(*os.File); ok {
		out, err := sshStream(f)
		if err != nil {
			return err
		}
		defer out.Close()
		output = out
	}
	return access.Bridge(ctx, in, output, socket, keys, fingerprint)
}

func sshStream(f *os.File) (*os.File, error) {
	fd, err := unix.FcntlInt(f.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), f.Name()), nil
}
