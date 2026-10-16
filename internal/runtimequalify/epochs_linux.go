//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// Metadata fingerprints intentionally exclude atime. They bind a preview to
// observed names/inodes/sizes/allocation/mtime/ctime, not to a content or backup hash.
type epochFingerprint struct {
	Path                             string
	Dev, Inode, Links                uint64
	Mode                             uint32
	Bytes, Blocks, Modified, Changed int64
	Metadata                         string
}

func (q *qualifier) inspectEpochs(ctx context.Context) (*epochs.Inventory, error) {
	return q.inspectEpochInventory(ctx, false)
}

// Import may retain old epoch snapshots without authorizing pruning/roll over
// them. The newest epoch must already use the supported 4.6 image.bin format.
func (q *qualifier) inspectImportEpochs(ctx context.Context) (*epochs.Inventory, error) {
	return q.inspectEpochInventory(ctx, true)
}

func (q *qualifier) inspectEpochInventory(ctx context.Context, history bool) (*epochs.Inventory, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	root, err := q.openPierDirectory("log")
	if err != nil {
		return nil, err
	}
	defer root.Close()
	first, err := q.scanEpochs(ctx, root, history)
	if err != nil {
		return nil, err
	}
	second, err := q.scanEpochs(ctx, root, history)
	if err != nil {
		return nil, err
	}
	if first.Digest != second.Digest {
		return nil, errors.New("epoch inventory changed during inspection")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	second.ObservedAt = time.Now().UTC()
	return second, nil
}

func (q *qualifier) openPierDirectory(leaf string) (*os.Root, error) {
	if leaf != "log" && leaf != "chk" {
		return nil, errors.New("unsupported pier directory")
	}
	if err := q.pier(); err != nil {
		return nil, err
	}
	// Anchor at the validated managed instance, then open each directory without
	// accepting symlinks. Root confines every later lookup, including rename races.
	root, err := os.OpenRoot(q.pierPath())
	if err != nil {
		return nil, errors.New("epoch storage unavailable")
	}
	keep := false
	defer func() {
		if !keep {
			root.Close()
		}
	}()
	rel := ""
	for _, part := range []string{".urb", leaf} {
		rel = filepath.Join(rel, part)
		if err := q.owned(filepath.Join(q.pierPath(), rel), os.ModeDir, false); err != nil {
			return nil, err
		}
		info, err := root.Lstat(part)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("unsafe epoch directory")
		}
		next, err := root.OpenRoot(part)
		if err != nil {
			return nil, errors.New("epoch directory unavailable")
		}
		actual, err := next.Stat(".")
		if err != nil || !os.SameFile(info, actual) {
			next.Close()
			return nil, errors.New("epoch directory changed")
		}
		root.Close()
		root = next
	}
	keep = true
	return root, nil
}

func boundedEntries(root *os.Root, limit int) ([]os.DirEntry, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, errors.New("epoch directory unreadable")
	}
	defer f.Close()
	entries, err := f.ReadDir(limit + 1)
	if (err != nil && !errors.Is(err, io.EOF)) || len(entries) > limit {
		return nil, errors.New("epoch directory entry limit exceeded or unreadable")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (q *qualifier) scanEpochs(ctx context.Context, root *os.Root, history ...bool) (*epochs.Inventory, error) {
	return scanPierEpochs(ctx, root, q.pierPath(), q.r.RunID, q.owned, history...)
}

func scanPierEpochs(ctx context.Context, root *os.Root, pier, id string, owned func(string, os.FileMode, bool) error, history ...bool) (*epochs.Inventory, error) {
	inv := &epochs.Inventory{Source: epochs.Source, PierID: id, Entries: []epochs.Entry{}}
	var manifest []epochFingerprint
	base := filepath.Join(pier, ".urb/log")
	rootInfo, err := root.Stat(".")
	if err != nil {
		return nil, errors.New("epoch root metadata unavailable")
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("epoch filesystem metadata unsupported")
	}
	fingerprint := func(r *os.Root, name, relative string, directory bool) (epochFingerprint, error) {
		if err := ctx.Err(); err != nil {
			return epochFingerprint{}, err
		}
		kind := os.FileMode(0)
		if directory {
			kind = os.ModeDir
		}
		if err := owned(filepath.Join(base, relative), kind, false); err != nil {
			return epochFingerprint{}, err
		}
		info, err := r.Lstat(name)
		if err != nil || info.Mode().Type() != kind {
			return epochFingerprint{}, errors.New("unsupported epoch entry type")
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Dev != rootStat.Dev || info.Size() < 0 || st.Blocks < 0 || st.Blocks > math.MaxInt64/512 || st.Nlink == 0 {
			return epochFingerprint{}, errors.New("unsafe epoch filesystem metadata")
		}
		return epochFingerprint{Path: relative, Dev: st.Dev, Inode: st.Ino, Links: st.Nlink, Mode: st.Mode, Bytes: info.Size(), Blocks: st.Blocks, Modified: info.ModTime().UnixNano(), Changed: st.Ctim.Nano()}, nil
	}
	rootFP, err := fingerprint(root, ".", ".", true)
	if err != nil {
		return nil, err
	}
	manifest = append(manifest, rootFP)
	entries, err := boundedEntries(root, epochs.MaxEpochs+2)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "data.mdb" || name == "lock.mdb" {
			fp, err := fingerprint(root, name, name, false)
			if err != nil {
				return nil, err
			}
			manifest = append(manifest, fp)
			continue
		}
		if _, err := epochs.Number(name); err != nil {
			return nil, errors.New("unrecognized epoch-log entry; review the format")
		}
		if len(inv.Entries) >= epochs.MaxEpochs {
			return nil, errors.New("epoch count limit exceeded")
		}
		fp, err := fingerprint(root, name, name, true)
		if err != nil {
			return nil, err
		}
		manifest = append(manifest, fp)
		dir, err := root.OpenRoot(name)
		if err != nil {
			return nil, errors.New("epoch directory unavailable")
		}
		e, files, scanErr := scanEpochDirectory(ctx, dir, name, fp, fingerprint, history...)
		dir.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		manifest = append(manifest, files...)
		inv.Entries = append(inv.Entries, e)
	}
	if len(inv.Entries) == 0 {
		return nil, errors.New("no epochs found in the initialized pier")
	}
	sort.Slice(inv.Entries, func(i, j int) bool {
		a, _ := epochs.Number(inv.Entries[i].Name)
		b, _ := epochs.Number(inv.Entries[j].Name)
		return a < b
	})
	for i := range inv.Entries {
		inv.Entries[i].Protected = i >= len(inv.Entries)-epochs.ProtectedSuffix
		if inv.Entries[i].LayoutVersion != 2 || inv.Entries[i].VereVersion != candidate.VereVersion {
			inv.Source = epochs.ImportSource
		}
	}
	if len(history) > 0 && history[0] {
		latest := inv.Entries[len(inv.Entries)-1]
		if latest.LayoutVersion != 2 || (latest.VereVersion != candidate.VereVersion && latest.VereVersion != "4.6") {
			return nil, errors.New("the latest epoch must be upgraded to Vere 4.6 before importing; historical epochs are preserved")
		}
	}
	data, err := json.Marshal(struct {
		Source, PierID string
		Manifest       []epochFingerprint
		Entries        []epochs.Entry
	}{inv.Source, inv.PierID, manifest, inv.Entries})
	if err != nil {
		return nil, errors.New("cannot fingerprint epoch inventory")
	}
	hash := sha256.Sum256(data)
	inv.Digest = hex.EncodeToString(hash[:])
	return inv, nil
}

func scanEpochDirectory(ctx context.Context, root *os.Root, name string, expected epochFingerprint, fingerprint func(*os.Root, string, string, bool) (epochFingerprint, error), history ...bool) (epochs.Entry, []epochFingerprint, error) {
	e := epochs.Entry{Name: name}
	fail := func() (epochs.Entry, []epochFingerprint, error) {
		return e, nil, errors.New("incomplete, changed or unsupported epoch layout")
	}
	actual, err := root.Stat(".")
	if err != nil {
		return fail()
	}
	st, ok := actual.Sys().(*syscall.Stat_t)
	if !ok || st.Dev != expected.Dev || st.Ino != expected.Inode {
		return fail()
	}
	entries, err := boundedEntries(root, epochs.MaxFilesPerEpoch)
	if err != nil {
		return e, nil, err
	}
	var manifest []epochFingerprint
	version, vere, database := false, false, false
	allowHistory := len(history) > 0 && history[0]
	filesPresent := map[string]bool{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return e, nil, err
		}
		file := entry.Name()
		filesPresent[file] = true
		fp, err := fingerprint(root, file, filepath.Join(name, file), false)
		if err != nil {
			return e, nil, err
		}
		if file == "epoc.txt" || file == "vere.txt" {
			if fp.Bytes > 128 {
				return fail()
			}
			f, err := root.OpenFile(file, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
			if err != nil {
				return fail()
			}
			info, statErr := f.Stat()
			if statErr != nil || !info.Mode().IsRegular() {
				f.Close()
				return fail()
			}
			fileStat, ok := info.Sys().(*syscall.Stat_t)
			if !ok || fileStat.Ino != fp.Inode || fileStat.Dev != fp.Dev {
				f.Close()
				return fail()
			}
			data, readErr := io.ReadAll(io.LimitReader(f, 129))
			f.Close()
			if readErr != nil || len(data) > 128 {
				return fail()
			}
			fp.Metadata = string(data)
			value := strings.TrimRight(string(data), " \t\r\n")
			if file == "epoc.txt" {
				version = value == "2"
				if value == "2" {
					e.LayoutVersion = 2
				} else if value == "1" {
					e.LayoutVersion = 1
				}
				if allowHistory {
					version = e.LayoutVersion == 1 || e.LayoutVersion == 2
				}
			} else {
				vere = value == candidate.VereVersion
				e.VereVersion = value
				if allowHistory {
					vere = historicalVereVersion.MatchString(value)
				}
			}
		}
		database = database || file == "data.mdb"
		if file == "image.bin" || allowHistory && (file == "north.bin" || file == "south.bin") {
			if fp.Bytes > math.MaxInt64-e.SnapshotBytes {
				return fail()
			}
			e.SnapshotBytes += fp.Bytes
		}
		if fp.Bytes > math.MaxInt64-e.LogicalBytes || fp.Blocks*512 > math.MaxInt64-e.AllocatedBytes {
			return fail()
		}
		e.LogicalBytes += fp.Bytes
		e.AllocatedBytes += fp.Blocks * 512
		e.FileCount++
		e.SharedFiles = e.SharedFiles || fp.Links > 1
		manifest = append(manifest, fp)
	}
	if !version || !vere || !database {
		return fail()
	}
	if allowHistory {
		if name != "0i0" && (e.SnapshotBytes <= 0 || (e.LayoutVersion == 1 && (!filesPresent["north.bin"] || !filesPresent["south.bin"])) || (e.LayoutVersion == 2 && !filesPresent["image.bin"])) {
			return fail()
		}
		if e.LayoutVersion == 1 && filesPresent["image.bin"] || e.LayoutVersion == 2 && (filesPresent["north.bin"] || filesPresent["south.bin"]) {
			return fail()
		}
	}
	data, err := json.Marshal(struct {
		Directory epochFingerprint
		Files     []epochFingerprint
	}{expected, manifest})
	if err != nil {
		return fail()
	}
	hash := sha256.Sum256(data)
	e.Digest = hex.EncodeToString(hash[:])
	return e, manifest, nil
}

var historicalVereVersion = regexp.MustCompile(`^(3\.[0-9]+|4\.[0-6])(?:-[0-9a-f]{7,40})?$`)
