package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
)

func recordingExport(ctx context.Context, args []string, config string) int {
	fail := func(err error) int { fmt.Fprintln(os.Stderr, err); return 1 }
	f := flag.NewFlagSet("recording-export", flag.ContinueOnError)
	in := service.RecordingExportInput{Representation: "original"}
	f.StringVar(&in.Name, "runner", "", "enrolled runner name")
	f.StringVar(&in.RunnerID, "expected-runner-id", "", "selected runner identity")
	f.StringVar(&in.InstanceID, "instance", "", "exact saved instance")
	f.StringVar(&in.JobID, "job", "", "exact immutable recording job ID")
	f.StringVar(&in.JobSHA256, "job-sha256", "", "exact saved job digest")
	f.BoolVar(&in.IncludePrivateData, "include-private-data", false, "explicit owner-private recording access")
	original := f.Bool("original", false, "explicit original bytes export choice")
	timingIndex := f.Bool("timing-index", false, "add a separate verified payload-free timing-index.jsonl; re-reads hash-pinned originals and refuses incomplete/oversized index output")
	output := f.String("output", "", "fresh absolute ZIP path in an existing owner-private physical directory")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !*original || in.Validate() != nil || *output == "" {
		return fail(errors.New("require an exact saved job, --original, --include-private-data and a fresh private --output path"))
	}
	if config == "" {
		var err error
		config, err = collection.DefaultDir()
		if err != nil {
			return fail(err)
		}
	}
	core, err := service.New(service.Options{ConfigDir: config})
	if err != nil {
		return fail(err)
	}
	var index recording.TimingIndexSummary
	digest, err := writeRecordingExport(*output, func(output io.Writer) error {
		if *timingIndex {
			var err error
			index, err = core.WriteRecordingBundleWithTimingIndex(ctx, in, output)
			return err
		}
		return core.WriteRecordingBundle(ctx, in, output)
	})
	if err != nil {
		return fail(err)
	}
	result := map[string]any{"status": "exported", "output": *output, "sha256": digest, "representation": "original", "retained_files_complete": true, "runtime_complete": false}
	if *timingIndex {
		result["timing_index"] = index
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fail(err)
	}
	return 0
}

// A partial file is retained on failure. The final name appears only after the
// archive's verified completion directory and file data are synced. Hard-link
// publication is no-overwrite, including a racing/symlink destination.
func writeRecordingExport(path string, write func(io.Writer) error) (string, error) {
	refused := errors.New("export requires a fresh absolute path in an existing physical owner-private directory; any partial output is retained with .partial suffix")
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) == "." || filepath.Base(path) == string(os.PathSeparator) {
		return "", refused
	}
	parent, name := filepath.Dir(path), filepath.Base(path)
	actual, err := filepath.EvalSymlinks(parent)
	if err != nil || actual != parent {
		return "", refused
	}
	private := func(info os.FileInfo) bool {
		st, ok := info.Sys().(*syscall.Stat_t)
		return ok && info.IsDir() && info.Mode().Perm() == 0700 && st.Uid == uint32(os.Geteuid())
	}
	before, err := os.Lstat(parent)
	if err != nil || !private(before) {
		return "", refused
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return "", refused
	}
	defer root.Close()
	bound := func() bool {
		resolved, err := filepath.EvalSymlinks(parent)
		if err != nil || resolved != parent {
			return false
		}
		current, e := os.Lstat(parent)
		if e != nil || !private(current) || !os.SameFile(before, current) {
			return false
		}
		dot, e := root.Stat(".")
		return e == nil && private(dot) && os.SameFile(before, dot)
	}
	if !bound() {
		return "", refused
	}
	if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return "", refused
	}
	partial := name + ".partial"
	file, err := root.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", refused
	}
	initial, err := file.Stat()
	if err != nil {
		file.Close()
		return "", refused
	}
	hash := sha256.New()
	err = write(io.MultiWriter(file, hash))
	err = errors.Join(err, file.Sync(), file.Close())
	if err != nil {
		return "", errors.New("recording export incomplete; private .partial output retained: " + err.Error())
	}
	saved, err := root.Lstat(partial)
	if err != nil || !saved.Mode().IsRegular() || saved.Mode().Perm() != 0600 || !os.SameFile(initial, saved) || !bound() {
		return "", refused
	}
	stat, ok := saved.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
		return "", refused
	}
	if err = root.Link(partial, name); err != nil {
		return "", refused
	}
	// Sync the new final directory entry before dropping the partial alias.
	syncRoot := func() error {
		d, e := root.Open(".")
		if e != nil {
			return e
		}
		return errors.Join(d.Sync(), d.Close())
	}
	if err = syncRoot(); err != nil {
		return "", err
	}
	if err = root.Remove(partial); err != nil {
		return "", err
	}
	if err = syncRoot(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
