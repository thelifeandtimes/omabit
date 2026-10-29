//go:build linux

package privatechain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// InspectRuntimeInputs never executes a file, contacts an engine, opens a chain
// through a database library, or changes a file/mode. The selected tree must be
// owned by the caller, canonical, and sealed (0500 directories, 0400/0500 files).
// It is a snapshot under the caller's UID, not isolation from that same UID.
// Mounts, links, unexpected files/directories and changes during hashing refuse.
func InspectRuntimeInputs(ctx context.Context, directory, expectedSHA, expectedFoundation string) (RuntimeInputInspection, error) {
	verifier, err := OpenRuntimeInputVerifier(ctx, directory, expectedSHA, expectedFoundation)
	if err != nil {
		return RuntimeInputInspection{}, err
	}
	defer verifier.Close()
	return verifier.Verify(ctx)
}

// The caller holds the selected root and brackets this snapshot with its current
// pathname binding. This helper never opens a different root or executes inputs.
func inspectRuntimeInputRoot(ctx context.Context, root *os.Root, expectedSHA, expectedFoundation string) (RuntimeInputInspection, error) {
	raw, err := readInputFile(ctx, root, "manifest.json", MaxRuntimeInputManifest, nil)
	if err != nil {
		return RuntimeInputInspection{}, err
	}
	m, err := DecodeRuntimeInputs(raw, expectedSHA, expectedFoundation)
	if err != nil {
		return RuntimeInputInspection{}, err
	}
	if err = verifyInputTree(ctx, root, m.Files); err != nil {
		return RuntimeInputInspection{}, err
	}
	// Re-read the manifest after the full traversal; no stale initial view passes.
	again, err := readInputFile(ctx, root, "manifest.json", MaxRuntimeInputManifest, nil)
	if err != nil {
		return RuntimeInputInspection{}, err
	}
	if string(again) != string(raw) {
		return RuntimeInputInspection{}, errRuntimeInputs
	}
	_, total, _ := m.validate()
	return RuntimeInputInspection{Format: RuntimeInputFormat, ManifestSHA256: expectedSHA, FoundationDigest: expectedFoundation, Files: len(m.Files), Bytes: total}, nil
}

func inputStat(info os.FileInfo) (*syscall.Stat_t, error) {
	s, ok := info.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != uint32(os.Geteuid()) {
		return nil, errRuntimeInputs
	}
	return s, nil
}

func openInputRoot(directory string) (*os.Root, error) {
	if !filepath.IsAbs(directory) || directory != filepath.Clean(directory) {
		return nil, errRuntimeInputs
	}
	real, err := filepath.EvalSymlinks(directory)
	if err != nil || real != directory {
		return nil, errRuntimeInputs
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode() != os.ModeDir|0500 {
		return nil, errRuntimeInputs
	}
	if _, err = inputStat(info); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errRuntimeInputs
	}
	actual, err := root.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		root.Close()
		return nil, errRuntimeInputs
	}
	return root, nil
}

// A device-number comparison alone misses bind mounts from the same filesystem.
// openat2 rejects mount and symlink traversal from the already selected root.
// Unsupported kernels refuse; there is no weaker open fallback.
func openInputFile(root *os.Root, name string, directory bool) (*os.File, error) {
	base, err := root.Open(".")
	if err != nil {
		return nil, errRuntimeInputs
	}
	defer base.Close()
	flags := uint64(unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC)
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat2(int(base.Fd()), name, &unix.OpenHow{Flags: flags, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
	if err != nil {
		return nil, errRuntimeInputs
	}
	return os.NewFile(uintptr(fd), name), nil
}

// If expectation is nil, return only bounded metadata bytes. Runtime files are
// streamed into their hashes and never accumulated or returned to callers.
func readInputFile(ctx context.Context, root *os.Root, name string, limit int64, expectation *RuntimeInputFile) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, errRuntimeInputs
	}
	st, err := inputStat(before)
	mode := os.FileMode(0400)
	if expectation != nil && expectation.Executable {
		mode = 0500
	}
	rootInfo, e := root.Stat(".")
	if e != nil {
		return nil, errRuntimeInputs
	}
	rs, e := inputStat(rootInfo)
	if err != nil || e != nil || !before.Mode().IsRegular() || before.Mode() != mode || st.Nlink != 1 || st.Dev != rs.Dev || before.Size() < 0 || before.Size() > limit {
		return nil, errRuntimeInputs
	}
	if expectation != nil && before.Size() != expectation.Bytes {
		return nil, errRuntimeInputs
	}
	f, err := openInputFile(root, name, false)
	if err != nil {
		return nil, errRuntimeInputs
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, errRuntimeInputs
	}
	reader := contextInputReader{ctx: ctx, reader: f}
	var raw []byte
	if expectation == nil {
		raw, err = io.ReadAll(io.LimitReader(reader, limit+1))
		if err != nil {
			return nil, err
		}
		if int64(len(raw)) != before.Size() {
			return nil, errRuntimeInputs
		}
	} else {
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(reader, expectation.Bytes+1))
		if e != nil {
			return nil, e
		}
		if n != expectation.Bytes || hex.EncodeToString(h.Sum(nil)) != expectation.SHA256 {
			return nil, errRuntimeInputs
		}
	}
	after, err := f.Stat()
	current, e := root.Lstat(name)
	if err != nil || e != nil || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Size() != before.Size() || current.Mode() != before.Mode() || after.ModTime() != before.ModTime() {
		return nil, errRuntimeInputs
	}
	last, e := inputStat(current)
	if e != nil || last.Nlink != 1 || last.Dev != st.Dev {
		return nil, errRuntimeInputs
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return raw, nil
}

func verifyInputTree(ctx context.Context, root *os.Root, files []RuntimeInputFile) error {
	expected := map[string]RuntimeInputFile{}
	directories := map[string]bool{".": true}
	for _, f := range files {
		expected[f.Path] = f
		for p := path.Dir(f.Path); p != "."; p = path.Dir(p) {
			directories[p] = true
		}
	}
	ri, err := root.Stat(".")
	if err != nil {
		return errRuntimeInputs
	}
	rs, err := inputStat(ri)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	err = fs.WalkDir(root.FS(), ".", func(name string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return errRuntimeInputs
		}
		info, err := root.Lstat(name)
		if err != nil {
			return errRuntimeInputs
		}
		st, err := inputStat(info)
		if err != nil || st.Dev != rs.Dev {
			return errRuntimeInputs
		}
		if d.IsDir() {
			if !directories[name] || info.Mode() != os.ModeDir|0500 {
				return errRuntimeInputs
			}
			dir, e := openInputFile(root, name, true)
			if e != nil {
				return e
			}
			return dir.Close()
		}
		if name == "manifest.json" {
			return nil
		}
		f, ok := expected[name]
		if !ok || seen[name] {
			return errRuntimeInputs
		}
		if _, err := readInputFile(ctx, root, name, f.Bytes, &f); err != nil {
			return err
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return errRuntimeInputs
	}
	return nil
}

type contextInputReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextInputReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
