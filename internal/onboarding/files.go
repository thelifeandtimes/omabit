package onboarding

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
)

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

func privateDirectory(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || strings.ContainsAny(dir, "\x00\r\n") {
		return errors.New("Setup directory must be an absolute clean path.")
	}
	for p := dir; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return errors.New("Setup directory cannot contain symlinks or files.")
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if p == "/" {
			break
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || info.Mode().Perm() != 0700 {
		return errors.New("Local setup storage must be owned by you with mode 0700.")
	}
	return nil
}

func readPrivate(path string, limit int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > limit {
		return nil, errors.New("Local setup file has unexpected ownership, permissions or size.")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("Local setup file exceeds its read limit.")
	}
	return data, nil
}

func writePrivate(path string, data []byte) error {
	if _, err := readPrivate(path, 1<<20); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".setup-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	err = errors.Join(err, f.Chmod(0600), f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func lockSetup(dir string) (func(), error) {
	path := filepath.Join(dir, ".lock")
	if _, err := readPrivate(path, 0); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("Local setup is already running. Wait for it to finish.")
	}
	return func() { f.Close() }, nil
}

func sshPath(path string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`).Replace(path) + `"`
}

func writeSSH(dir, hostPublic string) (client.SSHConfig, error) {
	known := filepath.Join(dir, "known_hosts")
	if err := writePrivate(known, []byte("[127.0.0.1]:22222 "+hostPublic+"\n")); err != nil {
		return client.SSHConfig{}, err
	}
	config := filepath.Join(dir, "ssh_config")
	text := fmt.Sprintf("Host omabit-local\n  HostName 127.0.0.1\n  Port 22222\n  User urbit-runner\n  IdentityFile %s\n  IdentityAgent none\n  IdentitiesOnly yes\n  UserKnownHostsFile %s\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  BatchMode yes\n", sshPath(filepath.Join(dir, "id_ed25519")), sshPath(known))
	if err := writePrivate(config, []byte(text)); err != nil {
		return client.SSHConfig{}, err
	}
	return client.SSHConfig{Host: "omabit-local", ConfigFile: config}, nil
}
