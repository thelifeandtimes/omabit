package recording

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

// WriteOriginalBundle streams an original-only ZIP, with bounded memory and no
// filesystem access. The caller owns authorization and a private unpublished
// destination. On failure the output is partial and must never be published as
// a completed bundle. fetch must not retry with a different manifest/session.
func WriteOriginalBundle(ctx context.Context, selected ExportRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error), output io.Writer) error {
	if result.Validate(selected) != nil || fetch == nil || output == nil || ctx.Err() != nil {
		return ErrSelection
	}
	z := zip.NewWriter(output)
	// Do not close on error: a partial stream must not gain a ZIP completion
	// directory. CLI adapters retain partial files under a distinct name.
	write := func(name string, data []byte) error {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0600)
		f, err := z.CreateHeader(header)
		if err == nil {
			_, err = f.Write(data)
		}
		return err
	}
	manifest, _ := json.MarshalIndent(result, "", "  ")
	if err := write("manifest.json", append(manifest, '\n')); err != nil {
		return err
	}
	for _, file := range result.Manifest.Files {
		header := &zip.FileHeader{Name: file.Name, Method: zip.Deflate}
		header.SetMode(0600)
		entry, err := z.CreateHeader(header)
		if err != nil {
			return err
		}
		hash := sha256.New()
		for index, digest := range file.Chunks {
			if err := ctx.Err(); err != nil {
				return err
			}
			request := ExportPartRequest{ExportRequest: selected, SessionID: result.Manifest.Metadata.SessionID, MetadataSHA256: result.Manifest.MetadataSHA256, FileName: file.Name, FileBytes: file.Bytes, ChunkIndex: index, ChunkSHA256: digest}
			part, err := fetch(ctx, request)
			if err != nil {
				return err
			}
			if part.Request != request || part.Validate() != nil {
				return ErrSelection
			}
			if _, err := entry.Write(part.Data); err != nil {
				return err
			}
			hash.Write(part.Data)
		}
		if hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
			return ErrCorrupt
		}
	}
	// The codec retains every frame's wall time, monotonic elapsed receive time,
	// stream, sequence, epoch and segment identity. This self-describing note is
	// an index contract, not a rendered terminal transcript or missing-byte claim.
	if err := write("timing-format.txt", []byte("Omabit recording codec1. Each frames*.obr segment retains its session/metadata digest, segment number, first sequence, receive epoch and prior digest in its header. Each frame retains sequence, wall Unix nanoseconds, elapsed receive nanoseconds, stream kind, exact payload length and linked SHA-256. Metadata versions3/4 link segments. Wall time may reverse; use sequence and receive epochs for order. Gap and exit payloads remain original canonical JSON. The manifest describes the verified prefix and any incomplete/corrupt tail. No output before attachment or full runtime coverage is asserted. Original terminal controls are data; do not replay them into a terminal merely to inspect this archive.\n")); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return z.Close()
}
