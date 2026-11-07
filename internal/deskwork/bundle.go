package deskwork

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

const bundleMagic = "OMABIT-DESK/1\n"
const MaxManifestBytes = 2 << 20
const MaxBundleBytes = MaxTreeBytes + MaxManifestBytes + int64(len(bundleMagic)) + 4

// A bundle is a bounded manifest followed by each file's exact declared bytes,
// in manifest order. No archive paths, modes, links or executable scripts are
// interpreted. It is staged outside a pier and cannot commit anything to Clay.
func WriteBundle(ctx context.Context, directory string, m Manifest, w io.Writer) error {
	if err := m.Validate(); err != nil {
		return err
	}
	before, err := Snapshot(directory)
	if err != nil {
		return err
	}
	if before.Digest != m.Digest {
		return errors.New("source changed before transfer")
	}
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if len(body) > MaxManifestBytes {
		return errors.New("source manifest too large")
	}
	if _, err = io.WriteString(w, bundleMagic); err != nil {
		return err
	}
	if err = binary.Write(w, binary.BigEndian, uint32(len(body))); err != nil {
		return err
	}
	if _, err = w.Write(body); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, item := range m.Files {
		if err = ctx.Err(); err != nil {
			return err
		}
		st, err := root.Lstat(item.Path)
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() || st.Size() != item.Bytes {
			return errors.New("source file changed before transfer")
		}
		f, err := root.Open(item.Path)
		if err != nil {
			return err
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(st, opened) {
			f.Close()
			return errors.New("source file replaced before transfer")
		}
		h := sha256.New()
		n, copyErr := io.CopyN(io.MultiWriter(w, h), &contextReader{ctx, f}, item.Bytes)
		closeErr := f.Close()
		if err = errors.Join(copyErr, closeErr); err != nil {
			return err
		}
		if n != item.Bytes || hex.EncodeToString(h.Sum(nil)) != item.SHA256 {
			return errors.New("source changed during transfer")
		}
	}
	after, err := Snapshot(directory)
	if err != nil {
		return err
	}
	if after.Digest != m.Digest {
		return errors.New("source changed during transfer")
	}
	return nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// StageBundle uses an exclusive new directory below the caller's private staging
// root. Only a complete, hash-verified, synced bundle is returned. Interrupted
// uploads remove their own staging directory and cannot affect a mounted desk.
func StageBundle(ctx context.Context, parent string, r io.Reader, expected string) (directory string, m Manifest, err error) {
	if !digestPattern.MatchString(expected) {
		return "", m, errors.New("invalid expected source digest")
	}
	st, err := os.Lstat(parent)
	if err != nil {
		return "", m, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return "", m, errors.New("staging root must be a private directory")
	}
	r = &contextReader{ctx, io.LimitReader(r, MaxBundleBytes+1)}
	magic := make([]byte, len(bundleMagic))
	if _, err = io.ReadFull(r, magic); err != nil || string(magic) != bundleMagic {
		return "", m, errors.New("invalid desk bundle")
	}
	var size uint32
	if err = binary.Read(r, binary.BigEndian, &size); err != nil {
		return "", m, err
	}
	if size == 0 || size > MaxManifestBytes {
		return "", m, errors.New("invalid manifest size")
	}
	body := make([]byte, size)
	if _, err = io.ReadFull(r, body); err != nil {
		return "", m, err
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err = d.Decode(&m); err != nil {
		return "", m, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || m.Validate() != nil || m.Digest != expected {
		return "", m, errors.New("source manifest mismatch")
	}
	directory, err = os.MkdirTemp(parent, ".desk-upload-")
	if err != nil {
		return "", m, err
	}
	staged := directory
	defer func() {
		if err != nil {
			os.RemoveAll(staged)
			directory = ""
		}
	}()
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", m, err
	}
	defer root.Close()
	dirs := map[string]bool{".": true}
	for _, item := range m.Files {
		dir := path.Dir(item.Path)
		if err = root.MkdirAll(dir, 0700); err != nil {
			return "", m, err
		}
		for p := dir; p != "."; p = path.Dir(p) {
			dirs[p] = true
		}
		var f *os.File
		f, err = root.OpenFile(item.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return "", m, err
		}
		h := sha256.New()
		_, copyErr := io.CopyN(io.MultiWriter(f, h), r, item.Bytes)
		syncErr := f.Sync()
		closeErr := f.Close()
		if err = errors.Join(copyErr, syncErr, closeErr); err != nil {
			return "", m, err
		}
		if hex.EncodeToString(h.Sum(nil)) != item.SHA256 {
			return "", m, fmt.Errorf("source hash mismatch: %s", item.Path)
		}
	}
	var trailing [1]byte
	if n, e := r.Read(trailing[:]); n != 0 || e != io.EOF {
		return "", m, errors.New("desk bundle contains trailing or unreadable bytes")
	}
	for dir := range dirs {
		f, e := root.Open(dir)
		if e != nil {
			err = e
			return "", m, err
		}
		err = errors.Join(f.Sync(), f.Close())
		if err != nil {
			return "", m, err
		}
	}
	// The manifest is deliberately not added inside the source tree. Persist it
	// in the operation receipt when publishing this stage into the managed cache.
	if f, e := os.Open(filepath.Dir(directory)); e != nil {
		err = e
		return "", m, err
	} else {
		err = errors.Join(f.Sync(), f.Close())
	}
	return directory, m, err
}
