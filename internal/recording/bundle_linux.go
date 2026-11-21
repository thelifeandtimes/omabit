//go:build linux

package recording

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"syscall"
	"time"
)

type exportReader struct {
	root *os.Root
	lock *os.File
	meta Metadata
	info os.FileInfo
}

func (e *exportReader) Close() { e.lock.Close(); e.root.Close() }

func (e *exportReader) unchanged(s *Store, id string) bool {
	after, err := s.root.Lstat(id)
	return err == nil && sameExportStat(e.info, after) && sameDirectory(e.root, after) && boundFile(e.root, "writer.lock", e.lock, 0)
}

func (s *Store) openExport(id string) (*exportReader, error) {
	if !APIHex(id, 32) {
		return nil, ErrSelection
	}
	r, err := childDirectory(s.root, id)
	if err != nil {
		return nil, err
	}
	lock, err := privateFile(r, "writer.lock", os.O_RDONLY, 0)
	if err != nil {
		r.Close()
		return nil, err
	}
	e := &exportReader{root: r, lock: lock}
	fail := func(err error) (*exportReader, error) { e.Close(); return nil, err }
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB) != nil {
		return fail(ErrStorage)
	}
	f, err := privateFile(r, "metadata.json", os.O_RDONLY, MaxMetadataBytes)
	if err != nil {
		return fail(err)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxMetadataBytes+1))
	f.Close()
	if err != nil {
		return fail(err)
	}
	e.meta, err = ReadMetadata(data)
	if err != nil || e.meta.SessionID != id {
		return fail(ErrCorrupt)
	}
	e.info, err = s.root.Lstat(id)
	if err != nil || !sameDirectory(r, e.info) {
		return fail(ErrStorage)
	}
	return e, nil
}

func exportFileUnchanged(r *os.Root, name string, file *os.File, before os.FileInfo) bool {
	after, err := file.Stat()
	return err == nil && sameExportStat(before, after) && boundFile(r, name, file, before.Size())
}

func sameExportStat(before, after os.FileInfo) bool {
	a, aok := before.Sys().(*syscall.Stat_t)
	b, bok := after.Sys().(*syscall.Stat_t)
	return aok && bok && a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Uid == b.Uid && a.Gid == b.Gid && a.Nlink == b.Nlink && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}

// ExportManifest never copies a session or rewrites originals. Holding the shared
// writer lock refuses active capture/continuation. Every part and complete file
// has a digest; assembling all parts must verify both before calling it complete.
func (s *Store) ExportManifest(ctx context.Context, id string) (ExportManifest, error) {
	var out ExportManifest
	e, err := s.openExport(id)
	if err != nil {
		return out, err
	}
	defer e.Close()
	scan, err := s.openSegments(e.root, e.meta)
	if err != nil {
		return out, err
	}
	defer scan.Close()
	initial := map[string]os.FileInfo{}
	for _, name := range append([]string{"metadata.json"}, scan.names...) {
		info, err := e.root.Lstat(name)
		if err != nil || !privateFileInfo(info, s.limits.MaxSessionBytes+MaxMetadataBytes) {
			return ExportManifest{}, ErrStorage
		}
		initial[name] = info
	}
	for name, info := range scan.verified {
		if !sameExportStat(info, initial[name]) {
			return ExportManifest{}, ErrStorage
		}
	}
	raw, _ := e.meta.Bytes()
	hash := sha256.Sum256(raw)
	out = ExportManifest{Version: 1, Metadata: e.meta, MetadataSHA256: hex.EncodeToString(hash[:]), Representation: "original", ChunkBytes: ExportChunkBytes, ReadAt: time.Now().UTC(), Basis: "exact retained encoded files under a shared writer lock; original terminal controls/private bytes require explicit owner choice; verified prefix and gaps are separate from runtime completeness; hashes detect changes, not source authenticity"}
	for {
		if err = ctx.Err(); err != nil {
			return ExportManifest{}, err
		}
		frame, err := scan.Next()
		if err != nil {
			switch {
			case err == io.EOF:
				out.Coverage.Tail = "snapshot-eof"
			case errors.Is(err, ErrIncomplete):
				out.Coverage.Tail = "incomplete"
			case errors.Is(err, ErrCorrupt), errors.Is(err, ErrInvalid):
				out.Coverage.Tail = "corrupt"
			default:
				return ExportManifest{}, err
			}
			break
		}
		c := &out.Coverage
		if c.VerifiedFrames == 0 {
			c.FirstUnixNano = frame.UnixNano
		}
		c.VerifiedFrames++
		c.LastUnixNano = frame.UnixNano
		switch frame.Kind {
		case Stdout, Stderr, PTYCombined:
			c.OutputBytes += uint64(len(frame.Data))
		case Gap:
			c.GapFrames++
		case Exit:
			c.ExitSaved = true
		}
	}
	var total int64
	for _, name := range append([]string{"metadata.json"}, scan.names...) {
		f, err := privateFile(e.root, name, os.O_RDONLY, s.limits.MaxSessionBytes+MaxMetadataBytes)
		if err != nil {
			return ExportManifest{}, err
		}
		item, err := hashExportFile(ctx, e.root, name, f)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return ExportManifest{}, errors.Join(err, closeErr)
		}
		if name != "metadata.json" {
			total += item.Bytes
		}
		out.Files = append(out.Files, item)
	}
	for name, before := range initial {
		after, err := e.root.Lstat(name)
		if err != nil || !sameExportStat(before, after) {
			return ExportManifest{}, ErrStorage
		}
	}
	if total > s.limits.MaxSessionBytes || !scan.unchanged() || !e.unchanged(s, id) || ctx.Err() != nil {
		return ExportManifest{}, ErrStorage
	}
	out.RetainedFilesComplete = true
	if err := out.Validate(); err != nil {
		return ExportManifest{}, err
	}
	return out, nil
}

func hashExportFile(ctx context.Context, root *os.Root, name string, f *os.File) (ExportFile, error) {
	before, err := f.Stat()
	if err != nil || before.Size() <= 0 || before.Size() > 1<<30 {
		return ExportFile{}, ErrStorage
	}
	out := ExportFile{Name: name, Bytes: before.Size(), Chunks: []string{}}
	whole := sha256.New()
	buf := make([]byte, ExportChunkBytes)
	for offset := int64(0); offset < before.Size(); offset += ExportChunkBytes {
		if err := ctx.Err(); err != nil {
			return ExportFile{}, err
		}
		part := buf[:min(int64(len(buf)), before.Size()-offset)]
		if _, err := f.ReadAt(part, offset); err != nil {
			return ExportFile{}, err
		}
		whole.Write(part)
		h := sha256.Sum256(part)
		out.Chunks = append(out.Chunks, hex.EncodeToString(h[:]))
	}
	if !exportFileUnchanged(root, name, f, before) {
		return ExportFile{}, ErrStorage
	}
	out.SHA256 = hex.EncodeToString(whole.Sum(nil))
	return out, nil
}

// ExportPart requires the caller's exact expected chunk hash from its saved
// manifest. It rehashes only this bounded chunk. Full-file/manifest verification
// belongs to assembly; a part is never itself a complete export.
func (s *Store) ExportPart(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
	if in.Validate() != nil || ctx.Err() != nil {
		return ExportPart{}, ErrSelection
	}
	e, err := s.openExport(in.SessionID)
	if err != nil {
		return ExportPart{}, err
	}
	defer e.Close()
	raw, _ := e.meta.Bytes()
	h := sha256.Sum256(raw)
	if hex.EncodeToString(h[:]) != in.MetadataSHA256 || e.meta.RunnerID != in.RunnerID || e.meta.InstanceID != in.InstanceID {
		return ExportPart{}, ErrSelection
	}
	f, err := privateFile(e.root, in.FileName, os.O_RDONLY, s.limits.MaxSessionBytes+MaxMetadataBytes)
	if err != nil {
		return ExportPart{}, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || before.Size() != in.FileBytes {
		return ExportPart{}, ErrSelection
	}
	offset := int64(in.ChunkIndex) * ExportChunkBytes
	data := make([]byte, min(int64(ExportChunkBytes), in.FileBytes-offset))
	if _, err = f.ReadAt(data, offset); err != nil || ctx.Err() != nil || !exportFileUnchanged(e.root, in.FileName, f, before) || !e.unchanged(s, in.SessionID) {
		return ExportPart{}, ErrStorage
	}
	if in.FileName == "metadata.json" && !bytes.Equal(raw, data) {
		return ExportPart{}, ErrCorrupt
	}
	out := ExportPart{Version: 1, Request: in, Data: data, ReadAt: time.Now().UTC()}
	if err := out.Validate(); err != nil {
		return ExportPart{}, err
	}
	return out, nil
}
