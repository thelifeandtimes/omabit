//go:build linux

package piercopy

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// WriteArchive is an internal stopped-handoff stream, not a historical backup API.
func WriteArchive(ctx context.Context, root *os.Root, to io.Writer) (Summary, error) {
	entries, err := inventory(ctx, root)
	if err != nil {
		return Summary{}, err
	}
	// Refuse before emitting any stream bytes; local copy's larger bound must
	// not silently change the existing cross-host transfer protocol.
	if digest(entries).Bytes > protocol.MaxPierBytes {
		return Summary{}, errors.New("pier exceeds the 100 GB host-handoff limit")
	}
	w := tar.NewWriter(to)
	for i, e := range entries {
		if err = ctx.Err(); err != nil {
			return Summary{}, err
		}
		h := &tar.Header{Name: e.Name, Mode: 0600, Size: e.Size, Typeflag: tar.TypeReg}
		if e.Directory {
			h.Mode = 0700
			h.Size = 0
			h.Typeflag = tar.TypeDir
		}
		if err = w.WriteHeader(h); err != nil {
			return Summary{}, err
		}
		if !e.Directory {
			entries[i].Hash, err = transfer(ctx, root, e, w)
			if err != nil {
				return Summary{}, err
			}
		}
	}
	if err = w.Close(); err != nil {
		return Summary{}, err
	}
	if err = unchanged(root, entries); err != nil {
		return Summary{}, err
	}
	return digest(entries), nil
}

// ReadArchive extracts only unique regular files/directories into a private empty
// root. Tar links, devices, traversal and implicit overwrites are never accepted.
func ReadArchive(ctx context.Context, from io.Reader, root *os.Root) (Summary, error) {
	initial, err := inventory(ctx, root)
	if err != nil || len(initial) != 1 {
		return Summary{}, errors.New("handoff target must be empty")
	}
	r := tar.NewReader(from)
	seen := map[string]bool{}
	var total int64
	for {
		if err = ctx.Err(); err != nil {
			return Summary{}, err
		}
		h, e := r.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return Summary{}, errors.New("invalid handoff archive")
		}
		name := h.Name
		if !utf8.ValidString(name) || len(name) > 4096 || strings.ContainsAny(name, "\x00\r\n\\") || path.Clean(name) != name || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || len(strings.Split(name, "/")) > MaxDepth+1 || seen[name] || len(seen) >= MaxEntries {
			return Summary{}, errors.New("unsafe or repeated handoff archive path")
		}
		seen[name] = true
		if h.Typeflag == tar.TypeDir {
			if name != "." {
				if err = root.Mkdir(name, 0700); err != nil {
					return Summary{}, errors.New("handoff directory creation failed")
				}
			}
			continue
		}
		if h.Typeflag != tar.TypeReg || name == "." || h.Size < 0 || h.Size > protocol.MaxPierBytes-total {
			return Summary{}, errors.New("unsupported handoff entry or byte limit")
		}
		f, e := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
		if e != nil {
			return Summary{}, errors.New("handoff file must be fresh")
		}
		_, err = io.CopyN(f, r, h.Size)
		err = errors.Join(err, f.Sync(), f.Close())
		if err != nil {
			return Summary{}, errors.New("incomplete handoff file")
		}
		total += h.Size
	}
	if !seen["."] {
		return Summary{}, errors.New("handoff archive has no root")
	}
	var extra [1]byte
	if n, e := from.Read(extra[:]); n != 0 || e != io.EOF {
		return Summary{}, errors.New("trailing handoff archive data")
	}
	summary, err := Measure(ctx, root)
	if err != nil {
		return summary, err
	}
	entries, err := inventory(ctx, root)
	if err != nil {
		return Summary{}, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Directory {
			f, e := root.Open(entries[i].Name)
			if e != nil {
				return Summary{}, e
			}
			if e = errors.Join(f.Sync(), f.Close()); e != nil {
				return Summary{}, e
			}
		}
	}
	return summary, nil
}
