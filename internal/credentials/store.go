// Package credentials keeps Eyre session material out of enrollment and output.
package credentials

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var ErrMissing = errors.New("no stored ship session; log in first")
var ErrStore = errors.New("credential storage failed or is unavailable; no automatic fallback was used")

const limit = 16384

type Store interface {
	Load(context.Context, string) ([]byte, error)
	Save(context.Context, string, []byte) error
	Delete(context.Context, string) error
}

func Scope(profileID, origin, sessionID string) string {
	sum := sha256.Sum256([]byte(profileID + "\x00" + origin + "\x00" + sessionID))
	return hex.EncodeToString(sum[:])
}
func validKey(key string) bool {
	data, err := hex.DecodeString(key)
	return err == nil && len(data) == 32 && key == strings.ToLower(key)
}

// File is an explicitly selected headless adapter. Its session files are private
// plaintext, not an encrypted keyring. There is no silent desktop fallback to it.
type File struct{ dir string }

func NewFile(dir string) (*File, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, ErrStore
	}
	parent := filepath.Dir(dir)
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return nil, ErrStore
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return nil, ErrStore
	}
	owner, ok := parentInfo.Sys().(*syscall.Stat_t)
	if !ok || !parentInfo.IsDir() || parentInfo.Mode().Perm() != 0700 || owner.Uid != uint32(os.Geteuid()) {
		return nil, ErrStore
	}
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, ErrStore
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, ErrStore
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode().Perm() != 0700 || st.Uid != uint32(os.Geteuid()) {
		return nil, ErrStore
	}
	return &File{dir: dir}, nil
}

func (s *File) open(key string) (*os.File, error) {
	if !validKey(key) {
		return nil, ErrStore
	}
	f, err := os.OpenFile(filepath.Join(s.dir, key), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, ErrStore
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, ErrStore
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 {
		f.Close()
		return nil, ErrStore
	}
	return f, nil
}

func (s *File) Load(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := s.open(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || len(data) > limit {
		clear(data)
		return nil, ErrStore
	}
	return data, nil
}

func (s *File) Save(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey(key) || len(data) == 0 || len(data) > limit {
		return ErrStore
	}
	// Each login gets a fresh random session ID. Refuse overwriting any existing
	// key, including links, rather than turning a stale login into a replacement.
	f, err := os.OpenFile(filepath.Join(s.dir, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return ErrStore
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		_ = os.Remove(filepath.Join(s.dir, key))
		return ErrStore
	}
	return s.sync()
}

func (s *File) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := s.open(key)
	if errors.Is(err, ErrMissing) {
		return nil
	}
	if err != nil {
		return err
	}
	f.Close()
	err = os.Remove(filepath.Join(s.dir, key))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ErrStore
	}
	return s.sync()
}

func (s *File) sync() error {
	f, err := os.Open(s.dir)
	if err != nil {
		return ErrStore
	}
	defer f.Close()
	if f.Sync() != nil {
		return ErrStore
	}
	return nil
}

type SecretService struct {
	run func(context.Context, []string, []byte) ([]byte, bool, error)
}

func NewSecretService() *SecretService { return &SecretService{run: runSecretTool} }

func (s *SecretService) Load(ctx context.Context, key string) ([]byte, error) {
	if !validKey(key) {
		return nil, ErrStore
	}
	data, missing, err := s.run(ctx, []string{"lookup", "application", "urbit-collector", "kind", "eyre-session", "scope", key}, nil)
	if missing {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, ErrStore
	}
	return bytes.TrimSpace(data), nil
}
func (s *SecretService) Save(ctx context.Context, key string, data []byte) error {
	if !validKey(key) || len(data) == 0 || len(data) > limit {
		return ErrStore
	}
	_, _, err := s.run(ctx, []string{"store", "--label=Urbit collector Eyre session", "application", "urbit-collector", "kind", "eyre-session", "scope", key}, data)
	if err != nil {
		return ErrStore
	}
	return nil
}
func (s *SecretService) Delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return ErrStore
	}
	_, missing, err := s.run(ctx, []string{"clear", "application", "urbit-collector", "kind", "eyre-session", "scope", key}, nil)
	if err != nil && !missing {
		return ErrStore
	}
	return nil
}

type cappedBuffer struct{ bytes.Buffer }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > limit {
		return 0, ErrStore
	}
	return b.Buffer.Write(p)
}

func runSecretTool(ctx context.Context, args []string, input []byte) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/secret-tool", args...)
	cmd.WaitDelay = time.Second
	cmd.Stdin = bytes.NewReader(input)
	var out, stderr cappedBuffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	if err != nil {
		clear(out.Bytes())
		var exit *exec.ExitError
		missing := errors.As(err, &exit) && exit.ExitCode() == 1 && stderr.Len() == 0 && ctx.Err() == nil
		return nil, missing, ErrStore
	}
	return out.Bytes(), false, nil
}
