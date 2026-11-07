//go:build linux

package deskwork

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const MaxCachedStages = 16
const MaxCachedSourceBytes int64 = 1 << 30

// The caller serializes cache authoring and fences deletion against any operation
// that references a stage. Entries are immutable, versioned disposable copies;
// no older reader may reinterpret their receipt format or transfer binding.
func privateStageDirectory(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("stage cache requires a canonical directory")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if real != path {
		return errors.New("linked stage cache directory")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !st.IsDir() || st.Mode().Perm() != 0700 || !ok || owner.Uid != uint32(os.Geteuid()) {
		return errors.New("stage cache must be private and owned by this account")
	}
	return nil
}
func PrepareStageCache(state string) (string, error) {
	if err := privateStageDirectory(state); err != nil {
		return "", err
	}
	root := filepath.Join(state, "desk-stages-v1")
	if err := os.Mkdir(root, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	if err := privateStageDirectory(root); err != nil {
		return "", err
	}
	return root, nil
}
func decodeCachedStage(body []byte) (CachedStage, error) {
	var s CachedStage
	if len(body) > MaxManifestBytes+16384 {
		return s, errors.New("stage receipt exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&s) != nil || d.Decode(new(any)) != io.EOF || s.Validate() != nil {
		return s, errors.New("invalid stage receipt")
	}
	canonical, _ := json.Marshal(s)
	if !bytes.Equal(canonical, body) {
		return s, errors.New("noncanonical stage receipt")
	}
	return s, nil
}
func stageMetadata(root, id string) (CachedStage, error) {
	var out CachedStage
	if !digestPattern.MatchString(id) {
		return out, errors.New("invalid source stage ID")
	}
	if err := privateStageDirectory(root); err != nil {
		return out, err
	}
	dir := filepath.Join(root, id)
	if err := privateStageDirectory(dir); err != nil {
		return out, err
	}
	out, err := cachedStageMetadata(dir)
	if err != nil {
		return out, err
	}
	pending, err := pendingStage(dir)
	if err != nil || out.Receipt.Intent.ID != id || pending != out.Receipt {
		return CachedStage{}, errors.New("stage receipt identity mismatch")
	}
	return out, nil
}
func cachedStageMetadata(dir string) (CachedStage, error) {
	var out CachedStage
	path := filepath.Join(dir, "receipt.json")
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return out, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return out, err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || !ok || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 || st.Size() > MaxManifestBytes+16384 {
		return out, errors.New("unsafe source stage receipt")
	}
	body, err := io.ReadAll(io.LimitReader(f, MaxManifestBytes+16385))
	if err != nil {
		return out, err
	}
	out, err = decodeCachedStage(body)
	if err != nil {
		return out, err
	}
	return out, nil
}

// ReadStage verifies cached source bytes before returning a path to an internal
// operation. That path is never a caller-selected host directory or a pier.
func ReadStage(ctx context.Context, root, id, actor string) (CachedStage, string, error) {
	s, err := stageMetadata(root, id)
	if err != nil {
		return CachedStage{}, "", err
	}
	if s.Receipt.Actor != actor {
		return CachedStage{}, "", errors.New("source stage belongs to another operator")
	}
	source := filepath.Join(root, id, "source")
	if err = privateStageDirectory(source); err != nil {
		return CachedStage{}, "", err
	}
	if err = ctx.Err(); err != nil {
		return CachedStage{}, "", err
	}
	actual, err := Snapshot(source)
	if err != nil || actual.Digest != s.Manifest.Digest {
		return CachedStage{}, "", errors.New("staged source changed or is unavailable")
	}
	if err = ctx.Err(); err != nil {
		return CachedStage{}, "", err
	}
	return s, source, nil
}
func stageCapacity(root string, additional int64) error {
	f, err := os.Open(root)
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(MaxCachedStages + 1)
	f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) >= MaxCachedStages {
		return errors.New("source stage cache is full; discard an unused stage")
	}
	var total int64
	for _, e := range entries {
		if !e.IsDir() || !digestPattern.MatchString(e.Name()) {
			return errors.New("source cache contains an incomplete or unknown entry; reconcile it before uploading")
		}
		s, err := stageMetadata(root, e.Name())
		if err != nil {
			return err
		}
		total += s.Receipt.Intent.Bytes
	}
	if total+additional > MaxCachedSourceBytes {
		return errors.New("source stage cache exceeds 1 GiB; discard an unused stage")
	}
	return nil
}

// CacheStage either verifies the exact prior receipt or atomically publishes a
// complete immutable stage. A failed upload removes only its newly created temp
// directory. A crash before publication can leave an explicit incomplete entry;
// it is never read as a usable stage or automatically replayed into Clay.
func CacheStage(ctx context.Context, root string, intent StageIntent, actor string, input io.Reader) (CachedStage, error) {
	var out CachedStage
	if intent.Validate() != nil || actor == "" || !settingText(actor, 128, false) {
		return out, errors.New("invalid source stage request")
	}
	if err := privateStageDirectory(root); err != nil {
		return out, err
	}
	cache, err := os.OpenRoot(root)
	if err != nil {
		return out, err
	}
	defer cache.Close()
	parent, err := cache.Open(".")
	if err != nil {
		return out, err
	}
	defer parent.Close()
	parentInfo, err := parent.Stat()
	if err != nil {
		return out, err
	}
	prior, _, err := ReadStage(ctx, root, intent.ID, actor)
	if err == nil {
		if prior.Receipt.Intent != intent {
			return out, errors.New("stage ID already records another transfer")
		}
		return prior, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return out, err
	}
	// A missing child of an existing stage is corruption, not a retryable absence.
	if _, e := os.Lstat(filepath.Join(root, intent.ID)); e == nil || !errors.Is(e, os.ErrNotExist) {
		return out, errors.New("existing stage is incomplete; preserve it for review")
	}
	if err = stageCapacity(root, intent.Bytes); err != nil {
		return out, err
	}
	temp := filepath.Join(root, ".pending-"+intent.ID)
	if err = os.Mkdir(temp, 0700); err != nil {
		return out, err
	}
	created, err := os.Lstat(temp)
	if err != nil {
		return out, err
	}
	defer func() {
		if named, e := cache.Lstat(filepath.Base(temp)); e == nil && os.SameFile(created, named) {
			_ = cache.RemoveAll(filepath.Base(temp))
		}
	}()
	planned := StageReceipt{Version: 1, Intent: intent, Actor: actor, CreatedAt: time.Now().UTC()}
	if err = writePendingStage(temp, planned); err != nil {
		return out, err
	}
	source, m, err := StageBundle(ctx, temp, input, intent.SourceDigest)
	if err != nil {
		return out, err
	}
	if err = intent.Check(m); err != nil {
		return out, err
	}
	if err = os.Rename(source, filepath.Join(temp, "source")); err != nil {
		return out, err
	}
	out = CachedStage{Receipt: planned, Manifest: m}
	if err = out.Validate(); err != nil {
		return CachedStage{}, err
	}
	body, _ := json.Marshal(out)
	receipt, err := os.OpenFile(filepath.Join(temp, "receipt.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return CachedStage{}, err
	}
	_, writeErr := receipt.Write(body)
	err = errors.Join(writeErr, receipt.Sync(), receipt.Close())
	if err != nil {
		return CachedStage{}, err
	}
	dir, err := os.Open(temp)
	if err != nil {
		return CachedStage{}, err
	}
	err = errors.Join(dir.Sync(), dir.Close())
	if err != nil {
		return CachedStage{}, err
	}
	if err = ctx.Err(); err != nil {
		return CachedStage{}, err
	}
	if err = privateStageDirectory(root); err != nil {
		return CachedStage{}, err
	}
	named, err := os.Lstat(root)
	if err != nil || !os.SameFile(parentInfo, named) {
		return CachedStage{}, errors.New("source cache directory changed during upload")
	}
	if err = unix.Renameat2(int(parent.Fd()), filepath.Base(temp), int(parent.Fd()), intent.ID, unix.RENAME_NOREPLACE); err != nil {
		return CachedStage{}, err
	}
	if err = parent.Sync(); err != nil {
		return CachedStage{}, err
	}
	return out, nil
}
