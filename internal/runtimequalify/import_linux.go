//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
)

func (q *qualifier) importBoundary(ctx context.Context) error {
	if q.d.managed || q.d.profile != ProfileImport || q.r.Profile != ProfileImport {
		return errors.New("pier-copy qualification requires a new disposable run")
	}
	if err := q.verifyEngine(ctx); err != nil {
		return err
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("pier transfer requires all prior containers removed")
		}
	}
	return nil
}

func (q *qualifier) importSource(ctx context.Context) error {
	if err := q.importBoundary(ctx); err != nil {
		return err
	}
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return errors.New("source boot requires 5 GiB available RAM and 8 GiB disk")
	}
	return q.boot(ctx, "boot")
}

func (q *qualifier) importCopy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := q.importBoundary(ctx); err != nil {
		return err
	}
	if err := q.pier(); err != nil {
		return err
	}
	root, err := os.OpenRoot(q.r.Directory)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Lstat("source"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("import source archive must be fresh")
	}
	source, err := root.OpenRoot("data")
	if err != nil {
		return err
	}
	before, err := piercopy.Measure(ctx, source)
	source.Close()
	if err != nil {
		return err
	}
	if before.Files == 0 {
		return errors.New("source pier contains no regular-file state")
	}
	_, disk, err := q.d.admission(q.r.Directory)
	if err != nil || disk < uint64(before.Bytes)+(8<<30) {
		return errors.New("pier copy requires source logical size plus 8 GiB free disk")
	}
	q.r.PierImport = &PierImportRecord{StartedAt: time.Now().UTC(), ContentDigest: before.Digest, Uncertain: true}
	if err := q.save(); err != nil {
		return err
	}
	if err := root.Rename("data", "source"); err != nil {
		return errors.New("cannot reserve the stopped source pier")
	}
	q.r.PierImport.SourceMoved = true
	if err := q.save(); err != nil {
		return err
	}
	if err := root.Mkdir("data", 0700); err != nil {
		return errors.New("cannot create private import destination")
	}
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	err = f.Sync()
	f.Close()
	if err != nil {
		return err
	}
	source, err = root.OpenRoot("source")
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := root.OpenRoot("data")
	if err != nil {
		return err
	}
	defer destination.Close()
	copied, err := piercopy.Copy(ctx, source, destination, func(files int, bytes int64) error {
		q.r.PierImport.CopiedFiles = files
		q.r.PierImport.CopiedBytes = bytes
		return q.save()
	})
	if err != nil {
		return err
	}
	if copied != before {
		return errors.New("copied source differed from the reviewed stopped manifest")
	}
	q.r.PierImport.CopyVerified = true
	return q.save()
}

func (q *qualifier) readyImport(ctx context.Context, index int) error {
	if err := q.ready(ctx, index); err != nil {
		return err
	}
	out, err := q.control(ctx, index, q.d.times.command, "exec", "/usr/bin/curl", "--disable", "--noproxy", "*", "--silent", "--show-error", "--fail", "--max-time", "5", "--max-filesize", "128", "http://127.0.0.1:8080/~/host")
	if err != nil || strings.TrimSpace(out) != "~zod" {
		return errors.New("copied-pier public identity was not the expected fake zod")
	}
	return nil
}

func (q *qualifier) importResume(ctx context.Context, kind string) error {
	if err := q.importBoundary(ctx); err != nil {
		return err
	}
	p := q.r.PierImport
	if p == nil || !p.CopyVerified || !p.Uncertain || (kind != "import-resume" && kind != "import-replacement") || (kind == "import-replacement" && !p.FirstResume) {
		return errors.New("copied pier is not ready for qualification resume")
	}
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return errors.New("import resume requires 5 GiB available RAM and 8 GiB disk")
	}
	if err := q.boot(ctx, kind); err != nil {
		return err
	}
	if kind == "import-resume" {
		p.FirstResume = true
	} else {
		p.ReplacementResume = true
	}
	return q.save()
}

func (q *qualifier) importFinish(ctx context.Context) error {
	if err := q.importBoundary(ctx); err != nil {
		return err
	}
	p := q.r.PierImport
	if p == nil || !p.CopyVerified || !p.FirstResume || !p.ReplacementResume {
		return errors.New("imported pier lacks both resume checks")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	source, err := os.OpenRoot(filepath.Join(q.r.Directory, "source"))
	if err != nil {
		return err
	}
	defer source.Close()
	observed, err := piercopy.Measure(ctx, source)
	if err != nil || observed.Digest != p.ContentDigest || observed.Files != p.CopiedFiles || observed.Bytes != p.CopiedBytes {
		return errors.New("stopped original pier changed during imported-pier resumes")
	}
	now := time.Now().UTC()
	p.SourceUnchanged = true
	p.FinishedAt = &now
	p.Uncertain = false
	return q.save()
}
