package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/remotesetup"
	"github.com/mattn/go-isatty"
	"golang.org/x/sys/unix"
)

func runDesktopAccess(ctx context.Context, args []string, out, errs io.Writer) int {
	f := flag.NewFlagSet("authorize-desktop", flag.ContinueOnError)
	f.SetOutput(errs)
	public := f.String("public-key", "", "desktop's Ed25519 public key")
	runner := f.String("expected-runner-id", "", "installed runner identity")
	role := f.String("role", "operator", "operator or observer")
	connection := f.String("ssh-connection", "", "authenticated OpenSSH connection tuple")
	revoke := f.Bool("revoke", false, "revoke this exact key without restarting management")
	ack := f.Bool("acknowledge-host-changes", false, "authorize only this management grant and restricted SSH key")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !*ack || *public == "" {
		fmt.Fprintln(errs, "require --public-key, --expected-runner-id, --ssh-connection and --acknowledge-host-changes")
		return 2
	}
	file, err := os.Open(*public)
	if err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	data, err := io.ReadAll(io.LimitReader(file, 8193))
	file.Close()
	if err != nil || len(data) > 8192 {
		fmt.Fprintln(errs, "invalid public key input")
		return 1
	}
	result, err := hostinstall.AuthorizeDesktop(ctx, data, hostinstall.DesktopAccessOptions{RunnerID: *runner, Role: *role, SSHConnection: *connection, Revoke: *revoke})
	json.NewEncoder(out).Encode(result)
	if err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	return 0
}

func runRemoteBootstrap(ctx context.Context, args []string, out, errs io.Writer) int {
	f := flag.NewFlagSet("remote-bootstrap", flag.ContinueOnError)
	f.SetOutput(errs)
	encoded := f.String("request", "", "base64 JSON public pairing request")
	login := f.String("login-user", "", "authenticated SSH login account")
	connection := f.String("ssh-connection", "", "OpenSSH connection tuple")
	resultPath := f.String("result", "", "private session result path")
	approve := f.Bool("approve", false, "explicit noninteractive approval for controlled provisioning")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || os.Geteuid() != 0 || len(*encoded) > 16384 || (!*approve && !isatty.IsTerminal(os.Stdin.Fd())) {
		fmt.Fprintln(errs, "remote bootstrap requires OS authorization and an interactive setup terminal")
		return 2
	}
	data, err := base64.StdEncoding.DecodeString(*encoded)
	var request remotesetup.Request
	if err == nil {
		d := json.NewDecoder(strings.NewReader(string(data)))
		d.DisallowUnknownFields()
		err = d.Decode(&request)
		var extra any
		if err == nil && d.Decode(&extra) != io.EOF {
			err = errors.New("extra request data")
		}
	}
	if err != nil || request.Validate() != nil {
		fmt.Fprintln(errs, "invalid pairing request")
		return 2
	}
	resultDir, err := openRemoteResult(*resultPath, *login)
	if err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	defer unix.Close(resultDir)
	input := bufio.NewReader(os.Stdin)
	confirm := func(message string) (bool, error) {
		fmt.Fprintln(out, "\n"+message)
		if *approve {
			return true, nil
		}
		fmt.Fprint(out, "Continue? [y/N] ")
		line, err := input.ReadString('\n')
		return strings.EqualFold(strings.TrimSpace(line), "y") || strings.EqualFold(strings.TrimSpace(line), "yes"), err
	}
	ctx, cancel := remotesetup.Deadline(ctx)
	defer cancel()
	result, setupErr := remotesetup.Bootstrap(ctx, request, *login, *connection, confirm, func(message string) { fmt.Fprintln(out, message) })
	envelope := struct {
		Result remotesetup.Result `json:"result"`
		Error  string             `json:"error,omitempty"`
	}{Result: result}
	if setupErr != nil {
		envelope.Error = setupErr.Error()
		fmt.Fprintln(errs, setupErr)
	}
	bytes, _ := json.Marshal(envelope)
	fd, err := unix.Openat(resultDir, "result.json", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err == nil {
		file := os.NewFile(uintptr(fd), "result.json")
		uid, e := loginUID(*login)
		if e == nil {
			e = file.Chown(int(uid), -1)
		}
		_, w := file.Write(bytes)
		err = errors.Join(e, w, file.Sync(), file.Close(), unix.Fsync(resultDir))
	}
	if err != nil {
		fmt.Fprintln(errs, "Result acknowledgement could not be saved; retry the same pairing.")
		return 1
	}
	if setupErr != nil {
		return 1
	}
	return 0
}

var remoteStage = regexp.MustCompile(`^/tmp/omabit-setup-[0-9a-f]{32}$`)

func loginUID(name string) (uint32, error) {
	if name == "root" {
		return 0, nil
	}
	return remotesetup.OwnerUID(name)
}

func openRemoteResult(path, login string) (int, error) {
	parent := filepath.Dir(path)
	if filepath.Base(path) != "result.json" || !remoteStage.MatchString(parent) || filepath.Clean(path) != path {
		return -1, errors.New("require the private remote setup result path")
	}
	uid, err := loginUID(login)
	if err != nil {
		return -1, err
	}
	if value := os.Getenv("SUDO_UID"); value != "" && value != strconv.FormatUint(uint64(uid), 10) {
		return -1, errors.New("setup login differs from the authorized caller")
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, parent, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return -1, err
	}
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Uid != uid || st.Mode&0777 != 0700 {
		unix.Close(fd)
		return -1, errors.New("unsafe setup result directory")
	}
	return fd, nil
}
