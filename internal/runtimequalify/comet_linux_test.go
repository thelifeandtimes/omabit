//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixtureComet = "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"

func cometFixture(t *testing.T) *fakeEngine {
	f := fixture(t)
	f.d.profile = ProfileComet
	f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded comet identity request")
		}
		for _, c := range f.containers {
			if (isCometKind(c.kind) || isCometRestart(c.kind) || isCometEpochRestart(c.kind) || isCometPrefixRuntime(c.kind)) && mapAt(c.doc, "State")["Running"] == true {
				port := "38101"
				if c.kind == "comet-resume" || c.kind == "loom-comet-resume" {
					port = "38102"
				}
				if c.kind == "comet-post-pack" {
					port = "38103"
				}
				if c.kind == "comet-post-meld" {
					port = "38104"
				}
				if f.d.hostPort != 0 {
					port = fmt.Sprint(f.d.hostPort)
				}
				if endpoint != "http://127.0.0.1:"+port+"/~/host" {
					t.Fatal("unbound comet host endpoint")
				}
				f.events = append(f.events, "comet-identity-"+c.kind)
				return fixtureComet, nil
			}
		}
		t.Fatal("comet probe without running owned container")
		return "", errors.New("missing")
	}
	return f
}

func TestCometCommandHasNoSuppliedIdentityAndResumeDoesNotCreate(t *testing.T) {
	for _, kind := range []string{"comet-boot", "comet-resume"} {
		entry, args := command(kind)
		if entry != "/bin/urbit" || args[len(args)-1] != "/urbit/comet" {
			t.Fatal("comet path changed")
		}
		for _, arg := range args {
			if arg == "-F" || arg == "-G" || arg == "-k" || arg == "-w" || arg == "-L" {
				t.Fatal("fake or supplied identity in comet command")
			}
		}
		contains := func(want string) bool {
			for _, arg := range args {
				if arg == want {
					return true
				}
			}
			return false
		}
		if !contains("--no-conn") || !contains("--no-dock") {
			t.Fatal("control/docking boundary changed")
		}
		if contains("-c") != (kind == "comet-boot") || contains("-B") != (kind == "comet-boot") {
			t.Fatal("comet bootstrap flags reused")
		}
	}
}

func TestCometQualificationPreservesIdentityAndCleansResources(t *testing.T) {
	f := cometFixture(t)
	var phases []string
	r, err := run(context.Background(), func(phase string) { phases = append(phases, phase) }, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || len(r.Containers) != 3 || len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatalf("comet workflow: %+v %v", r, err)
	}
	if !reflect.DeepEqual(phases, []string{"engine", "image", "pill", "probe", "network", "comet-boot", "comet-resume", "cleanup"}) {
		t.Fatal("unexpected comet phases", phases)
	}
	if r.Comet == nil || r.Comet.Ship != fixtureComet || r.Comet.Uncertain || r.Comet.FinishedAt == nil || !r.Comet.BootCompleted || !r.Comet.ResumeCompleted {
		t.Fatal("comet evidence missing")
	}
	if r.Containers[1].HostEndpoint == r.Containers[2].HostEndpoint {
		t.Fatal("replacement binding was not rediscovered")
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReport(r.RunID); err != nil {
		t.Fatal("completed comet report rejected", err)
	}
	assertReport(t, r)
}

func TestCometWrongIdentityAndTimeoutStayQuarantined(t *testing.T) {
	for _, fault := range []string{"fake", "changed", "body", "timeout", "resume-admission"} {
		t.Run(fault, func(t *testing.T) {
			f := cometFixture(t)
			read := f.d.hostIdentity
			f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
				if fault == "fake" {
					return "~zod", nil
				}
				if fault == "body" {
					return "PRIVATE-UNTRUSTED-BODY", errors.New("PRIVATE-UNTRUSTED-ERROR")
				}
				if fault == "timeout" {
					return "", errHostNotReady
				}
				if fault == "changed" && strings.Contains(endpoint, "38102") {
					return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
				}
				ship, err := read(ctx, endpoint)
				return ship, err
			}
			// The dependencies passed to run are copied; drive the admission change
			// from the observed identity events rather than rewriting the copy.
			if fault == "resume-admission" {
				f.d.admission = func(string) (uint64, uint64, error) {
					for _, event := range f.events {
						if event == "comet-identity-comet-boot" {
							return 0, 8 << 30, nil
						}
					}
					return 5 << 30, 8 << 30, nil
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Comet == nil || !r.Comet.Uncertain || r.Comet.FinishedAt != nil || strings.Contains(r.Failure, "PRIVATE") {
				t.Fatalf("unsafe comet outcome: %+v %v", r, err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("uncertain comet admitted to history")
			}
		})
	}
}

func TestCometRejectsUnqualifiedPublicationAndManagedContext(t *testing.T) {
	f := cometFixture(t)
	f.d.managed = true
	if r, err := run(context.Background(), nil, f.d); err == nil || r.RunID != "" || len(f.calls) != 0 {
		t.Fatal("managed comet qualification dispatched")
	}
	for _, fault := range []string{"wildcard", "ames-publication"} {
		t.Run(fault, func(t *testing.T) {
			f := cometFixture(t)
			f.d.hostIdentity = func(context.Context, string) (string, error) { t.Fatal("unsafe publication probed"); return "", nil }
			f.editInspect = func(kind string, doc map[string]any) {
				if !isCometKind(kind) {
					return
				}
				bindings := mapAt(mapAt(doc, "HostConfig"), "PortBindings")
				if fault == "wildcard" {
					bindings["8080/tcp"] = []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": ""}}
				} else {
					bindings["34343/udp"] = []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": "34343"}}
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed {
				t.Fatal("unsafe comet publication accepted")
			}
		})
	}
}

func TestCometHistoryRequiresExplicitOutcomeEvidence(t *testing.T) {
	for _, field := range []string{"uncertain", "ship", "finished_at", "boot_completed", "resume_completed"} {
		t.Run(field, func(t *testing.T) {
			f := cometFixture(t)
			r, err := run(context.Background(), nil, f.d)
			if err != nil {
				t.Fatal(err)
			}
			var value map[string]any
			data, _ := os.ReadFile(r.ReportPath)
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatal(err)
			}
			delete(value["comet"].(map[string]any), field)
			data, _ = json.Marshal(value)
			if err := os.WriteFile(r.ReportPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("incomplete comet evidence accepted")
			}
		})
	}
}
