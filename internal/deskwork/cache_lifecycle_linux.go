//go:build linux

package deskwork

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func writePendingStage(dir string, r StageReceipt) error {
	if err := r.Validate(); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "intent.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(r)
	_, writeErr := f.Write(body)
	if err = errors.Join(writeErr, f.Sync(), f.Close()); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	return errors.Join(d.Sync(), d.Close())
}
func pendingStage(dir string) (StageReceipt, error) {
	var out StageReceipt
	if err := privateStageDirectory(dir); err != nil {
		return out, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "intent.json"), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return out, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return out, err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !ok || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 || info.Size() > 16384 {
		return out, errors.New("unsafe staged transfer intent")
	}
	body, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return out, err
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&out) != nil || d.Decode(new(any)) != io.EOF || out.Validate() != nil {
		return out, errors.New("invalid staged transfer intent")
	}
	canonical, _ := json.Marshal(out)
	if !bytes.Equal(canonical, body) {
		return out, errors.New("noncanonical staged transfer intent")
	}
	return out, nil
}

// DiscardStage removes only the selected operator's cached copy. The caller must
// serialize authoring and first refuse stages referenced by unresolved effects.
// The retained intent supports cleanup of a crashed upload and a partly completed
// discard. Original source directories and piers are never arguments to this API.
func DiscardStage(root, id, actor, expectedDigest string, guard func(StageReceipt) error) error {
	if !digestPattern.MatchString(id) || !digestPattern.MatchString(expectedDigest) || actor == "" {
		return errors.New("invalid source discard selection")
	}
	if err := privateStageDirectory(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	cache, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer cache.Close()
	parent, err := cache.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	parentInfo, err := parent.Stat()
	if err != nil {
		return err
	}
	var present []string
	for _, name := range []string{id, ".discard-" + id, ".pending-" + id} {
		if _, e := cache.Lstat(name); errors.Is(e, os.ErrNotExist) {
			continue
		} else if e != nil {
			return e
		}
		present = append(present, name)
	}
	if len(present) > 1 {
		return errors.New("multiple stage entries require review")
	}
	names := []string{}
	for _, name := range present {
		r, e := pendingStage(filepath.Join(root, name))
		if e != nil {
			// A crash between mkdir and the first intent write can leave only an
			// empty, private pending directory. Remove never traverses children;
			// malformed metadata or any source content must be preserved.
			if name != id && errors.Is(e, os.ErrNotExist) && privateStageDirectory(filepath.Join(root, name)) == nil {
				named, e := os.Lstat(root)
				if e != nil || !os.SameFile(parentInfo, named) {
					return errors.New("source cache changed before discard")
				}
				if err = cache.Remove(name); err == nil {
					return parent.Sync()
				}
			}
			return e
		}
		if r.Intent.ID != id {
			return errors.New("stage discard identity mismatch")
		}
		if full, e := cachedStageMetadata(filepath.Join(root, name)); e == nil {
			if full.Receipt != r {
				return errors.New("stage metadata disagrees with retained intent")
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if r.Actor != actor || r.Intent.SourceDigest != expectedDigest {
			return errors.New("stage discard selection or operator changed")
		}
		if guard != nil {
			if err = guard(r); err != nil {
				return err
			}
		}
		names = append(names, name)
	}
	for _, name := range names {
		named, e := os.Lstat(root)
		if e != nil || !os.SameFile(parentInfo, named) {
			return errors.New("source cache changed before discard")
		}
		r, e := pendingStage(filepath.Join(root, name))
		if e != nil || r.Intent.ID != id || r.Intent.SourceDigest != expectedDigest || r.Actor != actor {
			return errors.New("source discard intent changed")
		}
		if name == id {
			if err = unix.Renameat2(int(parent.Fd()), id, int(parent.Fd()), ".discard-"+id, unix.RENAME_NOREPLACE); err != nil {
				return err
			}
			name = ".discard-" + id
			if err = parent.Sync(); err != nil {
				return err
			}
		}
		if err = removeStageContents(cache, name); err != nil {
			return err
		}
		if err = parent.Sync(); err != nil {
			return err
		}
	}
	return nil
}

// Delete the intent last so a crash during a large recursive source removal still
// leaves an authenticated recovery selection. After intent removal the directory
// is empty, which DiscardStage can remove without traversing unbound content.
func removeStageContents(cache *os.Root, name string) error {
	stage, err := cache.OpenRoot(name)
	if err != nil {
		return err
	}
	defer stage.Close()
	dir, err := stage.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(5)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 4 {
		return errors.New("unknown stage contents")
	}
	for _, entry := range entries {
		n := entry.Name()
		if n != "intent.json" && n != "source" && n != "receipt.json" && !strings.HasPrefix(n, ".desk-upload-") {
			return errors.New("unknown stage contents")
		}
	}
	for _, entry := range entries {
		if entry.Name() != "intent.json" {
			if err = stage.RemoveAll(entry.Name()); err != nil {
				return err
			}
		}
	}
	if err = dir.Sync(); err != nil {
		return err
	}
	if err = stage.Remove("intent.json"); err != nil {
		return err
	}
	if err = dir.Sync(); err != nil {
		return err
	}
	return cache.Remove(name)
}
