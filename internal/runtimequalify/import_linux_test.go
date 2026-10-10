//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func importFixture(t *testing.T) *fakeEngine {
	f := fixture(t)
	f.d.profile = ProfileImport
	f.d.admission = func(string) (uint64, uint64, error) { return 6 << 30, 20 << 30, nil }
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--max-filesize 128") {
			if !strings.HasSuffix(joined, "http://127.0.0.1:8080/~/host") || !strings.Contains(joined, "--disable --noproxy *") {
				t.Fatal("unsafe identity probe")
			}
			return "~zod", nil
		}
		out, err := execute(ctx, args, env)
		if err == nil && len(args) > 6 && args[4] == "container" && args[5] == "start" {
			for _, c := range f.containers {
				if c.id == args[6] {
					file := filepath.Join(filepath.Dir(args[3]), "data/fake-zod/.urb/state")
					if c.kind == "boot" {
						if err := os.WriteFile(file, []byte("original stopped state"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if c.kind == "import-resume" {
						if err := os.WriteFile(file, []byte("resumed target state"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if c.kind == "import-replacement" {
						data, err := os.ReadFile(file)
						if err != nil || string(data) != "resumed target state" {
							t.Fatal("replacement did not resume target")
						}
					}
				}
			}
		}
		return out, err
	}
	return f
}

func TestImportCopiesAndResumesOnlyFreshTarget(t *testing.T) {
	f := importFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || len(r.Containers) != 4 {
		t.Fatalf("import run: %v %+v", err, r)
	}
	p := r.PierImport
	if p == nil || p.Uncertain || p.FinishedAt == nil || !p.CopyVerified || !p.FirstResume || !p.ReplacementResume || !p.SourceUnchanged || p.CopiedFiles != 1 {
		t.Fatal("incomplete transfer evidence")
	}
	data, err := os.ReadFile(filepath.Join(r.Directory, "source/fake-zod/.urb/state"))
	if err != nil || string(data) != "original stopped state" {
		t.Fatal("source modified")
	}
	for _, kind := range []string{"import-resume", "import-replacement"} {
		_, args := command(kind)
		for _, a := range args {
			if a == "-F" || a == "-B" || a == "-c" {
				t.Fatal("copied pier reinitialized")
			}
		}
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReport(r.RunID); err != nil {
		t.Fatal("completed transfer report rejected", err)
	}
	assertReport(t, r)
}

func TestImportResumeFailureRetainsQuarantine(t *testing.T) {
	f := importFixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) > 6 && args[4] == "container" && args[5] == "start" {
			for _, c := range f.containers {
				if c.id == args[6] && c.kind == "import-resume" {
					return "", errors.New("interrupted imported runtime start")
				}
			}
		}
		return execute(ctx, args, env)
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || !r.CleanupComplete || r.PierImport == nil || !r.PierImport.Uncertain || r.PierImport.FinishedAt != nil {
		t.Fatal("failed resume lost quarantine", err)
	}
	for _, c := range r.Containers {
		if c.Kind == "import-replacement" {
			t.Fatal("continued after uncertain resume")
		}
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReport(r.RunID); err == nil {
		t.Fatal("uncertain import history accepted")
	}
}

func TestImportHistoryRequiresExplicitCompletion(t *testing.T) {
	for _, field := range []string{"uncertain", "copy_verified", "source_unchanged", "first_resume", "replacement_resume", "finished_at", "containers"} {
		t.Run(field, func(t *testing.T) {
			f := importFixture(t)
			r, err := run(context.Background(), nil, f.d)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			raw, _ := os.ReadFile(r.ReportPath)
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			if field == "containers" {
				data["containers"] = []any{}
			} else {
				delete(data["pier_import"].(map[string]any), field)
			}
			raw, _ = json.Marshal(data)
			if err := os.WriteFile(r.ReportPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("missing completion evidence accepted", field)
			}
		})
	}
}
