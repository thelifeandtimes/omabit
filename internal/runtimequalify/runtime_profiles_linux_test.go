//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestRuntimeProfileMissingCorruptAndUnknownFailBeforeExecution(t *testing.T) {
	for _, mode := range []string{"missing", "corrupt", "symlink", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state.RuntimeProfile = candidate.JSONFixProfile
			f.d.runtimeProfile = state.RuntimeProfile
			q := &qualifier{d: f.d}
			if mode != "missing" {
				if err := os.MkdirAll(filepath.Dir(q.profileArchive()), 0700); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "corrupt":
					if err := os.WriteFile(q.profileArchive(), []byte("wrong artifact"), 0600); err != nil {
						t.Fatal(err)
					}
				case "symlink":
					if err := os.Symlink("missing", q.profileArchive()); err != nil {
						t.Fatal(err)
					}
				case "unknown":
					state.RuntimeProfile = "unreviewed-build"
				}
			}
			_, _, err := applyManaged(t.Context(), "start", state, persist, f.d)
			if err == nil {
				t.Fatal("unverified runtime started")
			}
			for _, event := range f.events {
				if event == "pull" || event == "version" || strings.HasPrefix(event, "create-") {
					t.Fatal("unverified runtime executed or fell back", event)
				}
			}
			if _, err := os.Stat(filepath.Join(managedPier(f, state), ".urb")); !os.IsNotExist(err) {
				t.Fatal("failure opened/bootstraped the pier", err)
			}
		})
	}
}

func TestRuntimeProfileEpochCompatibilityRefusesImplicitTransition(t *testing.T) {
	f, state, _ := stoppedManagedFixture(t)
	base := epochFixture(t, f, state, "0i0", "0i10")
	if err := os.WriteFile(filepath.Join(base, "0i10/image.bin"), []byte("synthetic-checkpoint"), 0600); err != nil {
		t.Fatal(err)
	}
	p, _ := candidate.Resolve(candidate.JSONFixProfile)
	if err := os.WriteFile(filepath.Join(base, "0i10/vere.txt"), []byte(p.VereVersion+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	q, err := managedSession(&state, func(ManagedState) error { return nil }, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.accountBoundary(false); err != nil {
		t.Fatal(err)
	}
	before := len(f.calls)
	if _, err := q.inspectImportEpochs(t.Context()); err == nil {
		t.Fatal("stock runtime admitted patched epoch")
	}
	q.d.runtimeProfile = candidate.JSONFixProfile
	if _, err := q.inspectImportEpochs(t.Context()); err != nil {
		t.Fatal("reviewed runtime refused own epoch history", err)
	}
	if err := os.WriteFile(filepath.Join(base, "0i10/vere.txt"), []byte(candidate.VereVersion+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := q.inspectImportEpochs(t.Context()); err == nil || !strings.Contains(err.Error(), "never downgrade") {
		t.Fatal("unprepared transition accepted", err)
	}
	if len(f.calls) != before {
		t.Fatal("epoch refusal executed a runtime")
	}
}

// This is source-routing coverage with a fake engine, not native qualification.
// The optional real archive lets the production hash verifier run unchanged.
func TestRuntimeProfileReviewedArchiveLifecycleRouting(t *testing.T) {
	source := os.Getenv("OMABIT_TEST_JSONFIX_ARCHIVE")
	if source == "" {
		t.Skip("exact reviewed archive required for routing fixture")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	f, state, persist := managedFixture(t)
	p, _ := candidate.Resolve(candidate.JSONFixProfile)
	state.RuntimeProfile, f.d.runtimeProfile = p.ID, p.ID
	f.imageID, f.version = p.ConfigDigest, "urbit "+p.VereVersion+"\n"
	q := &qualifier{d: f.d}
	if err := os.MkdirAll(filepath.Dir(q.profileArchive()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(q.profileArchive(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || !state.Initialized {
		t.Fatal("first start", err, obs)
	}
	state, obs, err = applyManaged(t.Context(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" {
		t.Fatal("stop", err, obs)
	}
	base := epochFixture(t, f, state, "0i0")
	if err := os.WriteFile(filepath.Join(base, "0i0/vere.txt"), []byte(p.VereVersion+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	state, obs, err = applyManaged(t.Context(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" {
		t.Fatal("replacement", err, obs)
	}
	if f.downloads != 1 {
		t.Fatal("initialized pier bootstrapped again")
	}
	state, obs, err = observeManaged(t.Context(), state, f.d)
	if err != nil || obs.State != "running" {
		t.Fatal("manager reopen", err, obs)
	}
	for _, c := range state.Runtime.Containers {
		if c.RuntimeProfile != p.ID || c.LastState != nil && c.LastState.ImageID != p.ConfigDigest {
			t.Fatal("launch or observation lost artifact", c)
		}
	}
	// Losing an artifact must still permit stopping an already running ship.
	if err := os.Remove(q.profileArchive()); err != nil {
		t.Fatal(err)
	}
	state, obs, err = applyManaged(t.Context(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" {
		t.Fatal("missing archive prevented stop", err, obs)
	}
	before := len(f.events)
	if _, _, err := applyManaged(t.Context(), "start", state, persist, f.d); err == nil {
		t.Fatal("missing retained artifact restarted")
	}
	for _, event := range f.events[before:] {
		if event == "pull" || strings.HasPrefix(event, "create-") {
			t.Fatal("missing artifact fell back", event)
		}
	}
}

func TestRuntimeProfileRecordingJobRetainsLaunchPin(t *testing.T) {
	j, _, _, _ := recordingJobFixture(t)
	p, _ := candidate.Resolve(candidate.JSONFixProfile)
	j.State.RuntimeProfile, j.State.Runtime.RuntimeProfile = p.ID, p.ID
	j.State.Runtime.Image, j.State.Runtime.RepoDigest = p.Image, p.Image
	j.State.Runtime.ImageID, j.State.Runtime.VereVersion = p.ConfigDigest, p.VereVersion
	for i := range j.State.Runtime.Containers {
		j.State.Runtime.Containers[i].RuntimeProfile = p.ID
	}
	raw, err := j.Bytes()
	if err != nil {
		t.Fatal("recording rejected reviewed binding", err)
	}
	got, err := ReadManagedRecordingJob(raw)
	if err != nil || got.State.RuntimeProfile != p.ID || got.State.Runtime.Containers[0].RuntimeProfile != p.ID {
		t.Fatal("recording dropped pin", err)
	}
	got.State.RuntimeProfile = ""
	if got.Validate() == nil {
		t.Fatal("recording accepted substituted runtime")
	}
}

func TestRuntimeProfileReviewedArchiveStagingAndPreflight(t *testing.T) {
	source := os.Getenv("OMABIT_TEST_JSONFIX_ARCHIVE")
	if source == "" {
		t.Skip("exact reviewed archive required for staging fixture")
	}
	f := fixture(t)
	p, _ := candidate.Resolve(candidate.JSONFixProfile)
	f.d.runtimeProfile, f.imageID = p.ID, p.ConfigDigest
	for n := 0; n < 2; n++ {
		got, err := installRuntimeProfile(t.Context(), p.ID, source, f.d)
		if err != nil || got != p {
			t.Fatal("stage/repeat", err)
		}
		if err := verifyRuntimeProfile(t.Context(), p.ID, f.d); err != nil {
			t.Fatal("preflight", err)
		}
	}
	f.editImage = func(image map[string]any) { image["Architecture"] = "arm64" }
	if err := verifyRuntimeProfile(t.Context(), p.ID, f.d); err == nil {
		t.Fatal("wrong platform passed preflight")
	}
	f.editImage = func(image map[string]any) { image["Id"] = candidate.ConfigDigest }
	if err := verifyRuntimeProfile(t.Context(), p.ID, f.d); err == nil {
		t.Fatal("stock substitution passed preflight")
	}
	f.editImage = nil
	q := &qualifier{d: f.d}
	if err := os.WriteFile(q.profileArchive(), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	before := len(f.calls)
	if _, err := installRuntimeProfile(t.Context(), p.ID, source, f.d); err == nil {
		t.Fatal("corrupt retained archive silently replaced")
	}
	if err := verifyRuntimeProfile(t.Context(), p.ID, f.d); err == nil {
		t.Fatal("corrupt archive passed preflight")
	}
	if len(f.calls) != before {
		t.Fatal("corrupt archive reached Docker")
	}
	if len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatal("artifact staging created ship resources")
	}
}

// Docker's containerd store cannot inspect a loaded OCI image by config digest.
// Exercise actual distinct lookup availability, rather than accepting either
// string in a fake that always returns the selected image.
func TestRuntimeProfileImageStoreLookup(t *testing.T) {
	p, _ := candidate.Resolve(candidate.JSONFixProfile)
	for _, tc := range []struct {
		name, profile, available string
		want                     []string
	}{
		{"containerd", p.ID, p.ManifestDigest, []string{p.ManifestDigest}},
		{"legacy", p.ID, p.ConfigDigest, []string{p.ManifestDigest, p.ConfigDigest}},
		{"unavailable", p.ID, "", []string{p.ManifestDigest, p.ConfigDigest}},
		{"official", "", candidate.Image, []string{candidate.Image}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fixture(t)
			f.d.runtimeProfile = tc.profile
			var references []string
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) != 7 || args[4] != "image" || args[5] != "inspect" {
					t.Fatalf("unexpected command: %q", args)
				}
				references = append(references, args[6])
				if args[6] != tc.available {
					return "", errors.New("No such image")
				}
				return f.execute(ctx, args, env)
			}
			q := imageLookupFixture(t, f)
			_, err := q.inspectImageReference(t.Context())
			if (err != nil) != (tc.available == "") || !reflect.DeepEqual(references, tc.want) {
				t.Fatalf("lookup: %v, %v; want %v", references, err, tc.want)
			}
		})
	}
}

func TestRuntimeProfileImageLookupStopsOnCancellation(t *testing.T) {
	f := fixture(t)
	f.d.runtimeProfile = candidate.JSONFixProfile
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	f.d.execute = func(context.Context, []string, []string) (string, error) {
		calls++
		cancel()
		return "", context.Canceled
	}
	q := imageLookupFixture(t, f)
	if _, err := q.inspectImageReference(ctx); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("cancelled lookup continued: calls=%d error=%v", calls, err)
	}
}

func imageLookupFixture(t *testing.T, f *fakeEngine) *qualifier {
	t.Helper()
	q := &qualifier{d: f.d}
	q.base = filepath.Join(f.d.home, ".local/share/urbit-runner/qualification")
	q.r.Directory = filepath.Join(q.base, "image-lookup")
	q.config = filepath.Join(q.r.Directory, "docker-config")
	for _, path := range append(q.storagePaths(), q.r.Directory, q.config) {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return q
}
