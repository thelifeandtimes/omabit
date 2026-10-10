//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func cometPrefixFixture(t *testing.T) *fakeEngine {
	f := cometEpochFixture(t)
	f.d.profile = ProfileCometPrefix
	// Seventeen containers and two recovery journals make report serialization
	// noticeably larger under -race than the base fixture's 40 ms ready budget.
	f.d.times.ready, f.d.times.stop = time.Second, time.Second
	f.d.times.cleanup, f.d.times.cleanupCommand = 2*time.Second, time.Second
	f.d.remove = func(root *os.Root, name string) error { return root.Remove(name) }
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err != nil || a[4] != "container" || a[5] != "start" {
			return out, err
		}
		for _, c := range f.containers {
			if c.id != a[6] || !(isNetworkedKind(c.kind) || isCometPrefixRecovery(c.kind)) {
				continue
			}
			checkpoint := filepath.Join(filepath.Dir(a[3]), "data/comet/.urb/chk/image.bin")
			if isCometPrefixRecovery(c.kind) {
				if _, err := os.Stat(checkpoint); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("replay started without checkpoint loss", err)
				}
			}
			if err := os.MkdirAll(filepath.Dir(checkpoint), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(checkpoint, []byte("fixture checkpoint, not a real memory image"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return out, err
	}
	return f
}

func TestCometPrefixExactDeletionRecoveryAndIdentity(t *testing.T) {
	f := cometPrefixFixture(t)
	var removed []string
	f.d.remove = func(root *os.Root, name string) error {
		base := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32))
		leaf := filepath.Base(root.Name())
		expected := filepath.Join(base, "data/comet/.urb", leaf)
		if leaf != "log" && leaf != "chk" {
			if leaf != "0i0" && leaf != "0i100" && leaf != "0i200" {
				t.Fatal("unexpected deletion root", leaf)
			}
			expected = filepath.Join(base, "data/comet/.urb/log", leaf)
		}
		requireFixtureRoot(t, root, expected)
		var stored Result
		data, err := os.ReadFile(filepath.Join(base, "report.json"))
		if err != nil || json.Unmarshal(data, &stored) != nil || stored.Comet == nil || !stored.Comet.Uncertain || len(stored.PrefixPrunes) == 0 {
			t.Fatal("mutation preceded durable comet intent")
		}
		p := stored.PrefixPrunes[len(stored.PrefixPrunes)-1]
		if filepath.Base(root.Name()) == "chk" {
			if p.Recovery == nil || !p.Recovery.Uncertain || name != "image.bin" {
				t.Fatal("checkpoint loss preceded intent")
			}
		} else if !p.Uncertain || len(p.Targets) == 0 {
			t.Fatal("unlink preceded prefix intent")
		}
		removed = append(removed, filepath.Join(filepath.Base(root.Name()), name))
		return root.Remove(name)
	}
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || len(r.Containers) != 17 || len(r.PrefixPrunes) != 2 || r.Comet.Uncertain || r.Comet.Ship != fixtureComet || len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatalf("comet prefix %+v %v", r, err)
	}
	want := []string{"0i0/data.mdb", "0i0/epoc.txt", "0i0/lock.mdb", "0i0/vere.txt", "log/0i0", "0i100/data.mdb", "0i100/epoc.txt", "0i100/image.bin", "0i100/lock.mdb", "0i100/vere.txt", "log/0i100", "chk/image.bin", "0i200/data.mdb", "0i200/epoc.txt", "0i200/image.bin", "0i200/lock.mdb", "0i200/vere.txt", "log/0i200", "chk/image.bin"}
	if !reflect.DeepEqual(removed, want) {
		t.Fatal("incorrect deletion set/order", removed)
	}
	for i, p := range r.PrefixPrunes {
		_, selected, remaining := prefixShape(i + 1)
		if p.Uncertain || p.FinishedAt == nil || len(p.Targets) != selected || len(p.After.Entries) != remaining || p.PostRestart == nil || p.Recovery == nil || p.Recovery.Uncertain || !p.Recovery.CheckpointRemoved || p.Recovery.PostRestart == nil || !sameRetainedPrefix(p.Before, p.After, selected) {
			t.Fatal("incomplete prefix outcome", p)
		}
	}
	if r.CometFinalEpochs == nil || !reflect.DeepEqual(epochNames(r.CometFinalEpochs), []string{"0i300", "0i400"}) {
		t.Fatal("final replacement lost newest-two inventory")
	}
	for _, c := range r.Containers {
		if !c.Removed || c.CreateUncertain || isNetworkedKind(c.Kind) && !c.HostHTTPVerified {
			t.Fatal("unverified container or identity", c)
		}
	}
	for _, args := range f.calls {
		call := strings.Join(args, " ")
		if strings.Contains(call, "container create ") && strings.Contains(call, "-comet-prefix-recovery-") {
			if !strings.Contains(call, "--network none") || !strings.HasSuffix(call, "play --full --loom 31 /urbit/comet") || strings.Contains(call, "--publish") || strings.Contains(call, "/artifacts/") {
				t.Fatal("recovery command escaped qualification", call)
			}
		}
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReports(context.Background()); err != nil {
		t.Fatal("complete comet prefix report refused", err)
	}
	assertReport(t, r)
}

func TestCometPrefixFailuresRetainQuarantineAndStopProgress(t *testing.T) {
	for _, fault := range []string{"between-epochs", "snapshot-unlink", "second-prefix", "changed-retained", "intent-save", "finish-save", "stale-first", "stale-second", "recovery-timeout", "missing-checkpoint", "recovery-memory", "restart-identity", "recovered-identity", "final-identity"} {
		t.Run(fault, func(t *testing.T) {
			f := cometPrefixFixture(t)
			base := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32))
			calls := 0
			f.d.remove = func(root *os.Root, name string) error {
				calls++
				at := filepath.Base(root.Name())
				if fault == "between-epochs" && at == "0i100" || fault == "snapshot-unlink" && at == "0i100" && name == "image.bin" || fault == "second-prefix" && at == "0i200" {
					return errors.New("injected unlink failure")
				}
				err := root.Remove(name)
				if fault == "changed-retained" && calls == 1 {
					if err := os.WriteFile(filepath.Join(base, "data/comet/.urb/log/0i300/data.mdb"), []byte("changed"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return err
			}
			once := false
			f.d.persist = func(r Result) error {
				if !once && len(r.PrefixPrunes) > 0 && (fault == "intent-save" || fault == "finish-save" && !r.PrefixPrunes[0].Uncertain) {
					once = true
					return errors.New("journal failure")
				}
				return (&qualifier{r: r}).save()
			}
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" {
					for _, c := range f.containers {
						if c.id == a[6] && c.kind == "comet-prefix-recovery-1" && fault == "recovery-timeout" {
							return "", context.DeadlineExceeded
						}
					}
				}
				out, err := execute(ctx, a, e)
				if err == nil && fault == "missing-checkpoint" && a[4] == "container" && a[5] == "wait" {
					for _, c := range f.containers {
						if c.id == a[6] && c.kind == "comet-prefix-recovery-1" {
							if err := os.Remove(filepath.Join(base, "data/comet/.urb/chk/image.bin")); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				return out, err
			}
			read := f.d.hostIdentity
			f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
				for _, c := range f.containers {
					if fault == "restart-identity" && c.kind == "comet-prefix-restart-1" || fault == "recovered-identity" && c.kind == "comet-prefix-recovered-2" || fault == "final-identity" && c.kind == "comet-resume" {
						return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
					}
				}
				return read(ctx, endpoint)
			}
			if fault == "recovery-memory" {
				f.d.admission = func(string) (uint64, uint64, error) {
					for _, event := range f.events {
						if event == "rm-comet-prefix-restart-1" {
							return 0, 8 << 30, nil
						}
					}
					return 5 << 30, 8 << 30, nil
				}
			}
			r, err := run(context.Background(), func(phase string) {
				if fault == "stale-first" && phase == "comet-prefix-delete-1" || fault == "stale-second" && phase == "comet-prefix-delete-2" {
					epoch := "0i0"
					if fault == "stale-second" {
						epoch = "0i200"
					}
					if err := os.WriteFile(filepath.Join(base, "data/comet/.urb/log", epoch, "data.mdb"), []byte("stale"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Comet == nil || !r.Comet.Uncertain || r.CometFinalEpochs != nil {
				t.Fatal("failed prefix outcome accepted", fault, err)
			}
			phase := "comet-prefix-delete-1"
			switch fault {
			case "second-prefix", "stale-second":
				phase = "comet-prefix-delete-2"
			case "recovery-timeout", "missing-checkpoint", "recovery-memory":
				phase = "comet-prefix-recovery-1"
			case "restart-identity":
				phase = "comet-prefix-restart-1"
			case "recovered-identity":
				phase = "comet-prefix-recovered-2"
			case "final-identity":
				phase = "comet-resume"
			}
			if len(r.Phases) < 2 || r.Phases[len(r.Phases)-2].Name != phase {
				t.Fatal("failure occurred outside intended boundary", fault, r.Phases, err)
			}
			if fault == "intent-save" && calls != 0 {
				t.Fatal("unlink preceded durable intent")
			}
			if fault == "recovery-memory" {
				if _, err := os.Stat(filepath.Join(base, "data/comet/.urb/chk/image.bin")); err != nil {
					t.Fatal("memory refusal removed checkpoint")
				}
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatal("cleanup erased uncertainty")
			}
			before := len(f.calls)
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			if next, err := run(context.Background(), nil, f.d); err == nil || next.RunID != "" || len(f.calls) != before {
				t.Fatal("unresolved history permitted another run")
			}
		})
	}
}

func TestCometPrefixHistoryRequiresCompleteEvidence(t *testing.T) {
	f := cometPrefixFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"final-inventory", "wrong-final", "comet", "prefix", "target", "prefix-marker", "recovery-marker", "recovery-restart", "container", "identity-check"} {
		t.Run(fault, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(original, &doc); err != nil {
				t.Fatal(err)
			}
			p := doc["prefix_prunes"].([]any)[1].(map[string]any)
			switch fault {
			case "final-inventory":
				delete(doc, "comet_final_epochs")
			case "wrong-final":
				doc["comet_final_epochs"] = doc["epoch_lifecycle"].(map[string]any)["final"]
			case "comet":
				delete(doc, "comet")
			case "prefix":
				delete(doc, "prefix_prunes")
			case "target":
				delete(p, "targets")
			case "prefix-marker":
				delete(p, "uncertain")
			case "recovery-marker":
				delete(p["recovery"].(map[string]any), "uncertain")
			case "recovery-restart":
				delete(p["recovery"].(map[string]any), "post_restart")
			case "container":
				doc["containers"] = doc["containers"].([]any)[:16]
			case "identity-check":
				delete(doc["containers"].([]any)[16].(map[string]any), "host_http_verified")
			}
			if err := os.WriteFile(r.ReportPath, []byte(encode(t, doc)), 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("incomplete history accepted", fault)
			}
		})
	}
}

func TestCometPrefixDestructionStaysInFreshQualification(t *testing.T) {
	for _, profile := range []string{ProfileHTTP, ProfileComet, ProfileCometEpochs, ProfileCometMaintenance} {
		q := &qualifier{d: dependencies{profile: profile}}
		if err := q.pruneWiderPrefix(context.Background(), 1); err == nil {
			t.Fatal("prefix deletion escaped profile")
		}
		for _, kind := range []string{"comet-roll-4", "comet-prefix-restart-1", "comet-prefix-recovery-1"} {
			if _, err := q.create(context.Background(), kind); err == nil || !strings.Contains(err.Error(), "not allowed") {
				t.Fatal("prefix container escaped profile", profile, kind, err)
			}
		}
	}
	f := cometPrefixFixture(t)
	f.d.managed = true
	if r, err := run(context.Background(), nil, f.d); err == nil || r.RunID != "" || len(f.calls) != 0 {
		t.Fatal("managed prefix qualification dispatched")
	}
	// A qualified profile alone cannot adopt an unregistered managed root.
	q := &qualifier{d: dependencies{managed: true, profile: ProfileComet, home: "/private"}}
	if err := q.stoppedPruneBoundary(context.Background()); err == nil {
		t.Fatal("unregistered managed root admitted")
	}
}
