package hostcompat

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

type pacmanRepositoryPackage struct {
	Name, Version, Architecture, Filename, PGPSig, SHA256 string
	Size                                                  int64
}

func pacmanArchiveName(name string, p nativePackage) bool {
	if path.Base(name) != name || strings.HasSuffix(name, ".sig") {
		return false
	}
	versions := []string{p.version}
	if _, v, ok := strings.Cut(p.version, ":"); ok {
		versions = append(versions, v)
	}
	for _, version := range versions {
		base := p.name + "-" + version + "-" + p.arch + ".pkg.tar"
		for _, suffix := range []string{"", ".zst", ".xz", ".gz", ".bz2", ".lz4", ".lrz", ".lzo", ".Z"} {
			if name == base+suffix {
				return true
			}
		}
	}
	return false
}

// Repository descriptions are lookup hints, never authentication. PGPSIG is a
// detached package signature; only current native PGP trust plus exact signed
// .PKGINFO and installed payload comparison can authenticate the package.
func (v *verifier) pacmanRepository(ctx context.Context, name string, p nativePackage) ([]pacmanRepositoryPackage, error) {
	before, err := digestFiles(v.fsys, []string{name}, maxIndex)
	if err != nil {
		return nil, err
	}
	b, err := v.run(ctx, "/usr/bin/bsdtar", []string{"-cf", "-", "--format=pax", "--", "@/" + name}, maxIndex)
	if err != nil || !verifyDigests(v.fsys, before, maxIndex) {
		return nil, errEvidence
	}
	tr := tar.NewReader(bytes.NewReader(b))
	var result []pacmanRepositoryPackage
	for n := 0; ; n++ {
		if n > 131072 {
			return nil, errEvidence
		}
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errEvidence
		}
		name := strings.TrimPrefix(h.Name, "./")
		if name == "" && h.Typeflag == tar.TypeDir {
			continue
		}
		if !fs.ValidPath(strings.TrimSuffix(name, "/")) {
			return nil, errEvidence
		}
		if path.Base(name) != "desc" || !strings.HasPrefix(name, p.name+"-") {
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size < 0 || h.Size > 256<<10 {
			return nil, errEvidence
		}
		data, err := io.ReadAll(io.LimitReader(tr, (256<<10)+1))
		if err != nil {
			return nil, errEvidence
		}
		fields, err := parsePacman(string(data))
		if err != nil {
			return nil, err
		}
		r := pacmanRepositoryPackage{Name: singleton(fields, "NAME"), Version: singleton(fields, "VERSION"), Architecture: singleton(fields, "ARCH"), Filename: singleton(fields, "FILENAME"), PGPSig: singleton(fields, "PGPSIG"), SHA256: singleton(fields, "SHA256SUM")}
		r.Size, _ = strconv.ParseInt(singleton(fields, "CSIZE"), 10, 64)
		if r.Name != p.name || r.Version != p.version || r.Architecture != p.arch {
			continue
		}
		if !pacmanArchiveName(r.Filename, p) || len(r.PGPSig) > 128<<10 {
			return nil, errEvidence
		}
		result = append(result, r)
		if len(result) > 8 {
			return nil, errEvidence
		}
	}
	return result, nil
}

func (v *verifier) embeddedPacmanSignature(ctx context.Context, p nativePackage) (string, error) {
	if v.store == nil {
		return "", errEvidence
	}
	dirs := append(append([]string{}, v.pacmanDBDirs...), "var/lib/pacman/sync")
	count := 0
	for _, dir := range dirs {
		entries, err := readDir(v.fsys, dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".db") {
				continue
			}
			count++
			if count > 64 {
				return "", errEvidence
			}
			records, err := v.pacmanRepository(ctx, dir+"/"+entry.Name(), p)
			if err != nil {
				continue
			}
			for _, record := range records {
				if record.PGPSig == "" {
					continue
				}
				sig, err := base64.StdEncoding.Strict().DecodeString(record.PGPSig)
				if err != nil || len(sig) == 0 || len(sig) > 96<<10 {
					continue
				}
				name, err := v.store(ctx, sig, ".sig")
				if err == nil {
					return name, nil
				}
			}
		}
	}
	return "", errEvidence
}
