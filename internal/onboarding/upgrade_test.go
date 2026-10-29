package onboarding

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type upgradeFixture struct {
	dir, runner             string
	deps                    upgradeDependencies
	binding                 upgradeBinding
	installed               []byte
	calls, prepares, probes int
	failure, errorProbe     error
	result                  *hostinstall.HostResult
}

func newUpgradeFixture(t *testing.T) *upgradeFixture {
	t.Helper()
	base := t.TempDir()
	f := &upgradeFixture{dir: filepath.Join(base, "setup"), runner: filepath.Join(base, "urbit-runner"), installed: []byte("previous")}
	if err := privateDirectory(f.dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.runner, []byte("candidate"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(filepath.Join(f.dir, "operator.pub"), []byte("fixture public key")); err != nil {
		t.Fatal(err)
	}
	f.binding = upgradeBinding{Owner: ownerhost.Profile{Version: 1, Name: "fixture", UID: 1001, GID: 1001, Home: "/home/fixture", Piers: "/srv/fixture/piers"}, RunnerID: "runner-fixture", EnrollmentSHA256: "connection", AccessSHA256: "access", Fingerprint: "operator", HostKey: "host"}
	f.deps = upgradeDependencies{
		binding:   func(context.Context, string, string) (upgradeBinding, error) { return f.binding, nil },
		probe:     func(context.Context, upgradeBinding) error { f.probes++; return f.errorProbe },
		installed: "fixture-installed",
		binary: func(path string) ([]byte, string, error) {
			if path == "fixture-installed" {
				return f.installed, "installed", nil
			}
			data, err := os.ReadFile(path)
			return data, "candidate", err
		},
		prepare: func(_ context.Context, path string) (rootless.Manifest, error) {
			f.prepares++
			return rootless.Manifest{}, os.Mkdir(path, 0700)
		},
		verifyBundle: func(path string) error { _, err := os.Stat(path); return err },
		elevate: func(_ context.Context, runner, bundle, public, expected string, _ func(Event)) (hostinstall.HostResult, error) {
			f.calls++
			if runner != filepath.Join(f.dir, "service-upgrade", "urbit-runner") || bundle != filepath.Join(f.dir, "service-upgrade", "rootless") || public != filepath.Join(f.dir, "service-upgrade", "operator.pub") || expected != f.binding.RunnerID {
				t.Fatal("unexpected installation scope")
			}
			data, err := os.ReadFile(runner)
			if err != nil {
				t.Fatal(err)
			}
			if f.failure != nil {
				return hostinstall.HostResult{}, f.failure
			}
			f.installed = data
			if f.result != nil {
				return *f.result, nil
			}
			return hostinstall.HostResult{Ready: true, Owner: f.binding.Owner.Name, UID: f.binding.Owner.UID, PierRoot: f.binding.Owner.Piers, RunnerID: f.binding.RunnerID, Mode: "development-managed", SSHPort: 22222, HostPublicKey: f.binding.HostKey, OperatorFingerprint: f.binding.Fingerprint}, nil
		}}
	return f
}
func (f *upgradeFixture) check() (UpgradeStatus, error) {
	return serviceUpgradeStatus(context.Background(), "fixture", f.dir, f.runner, f.deps)
}
func (f *upgradeFixture) apply(s UpgradeStatus, emit func(UpgradeEvent)) (UpgradeStatus, error) {
	return upgradeServices(context.Background(), "fixture", f.dir, f.runner, s.RunnerID, s.TargetSHA256, f.deps, emit)
}
func TestServiceUpgradeReadOnlyCheckAndVerifiedNoOp(t *testing.T) {
	f := newUpgradeFixture(t)
	before, _ := os.ReadDir(f.dir)
	s, err := f.check()
	after, _ := os.ReadDir(f.dir)
	if err != nil || !s.Available || s.Pending || len(before) != len(after) || f.calls+f.probes+f.prepares != 0 {
		t.Fatal(s, err)
	}
	var complete bool
	s, err = f.apply(s, func(e UpgradeEvent) {
		if e.Step == "upgrade" && e.Status == "complete" {
			complete = true
		}
	})
	if err != nil || s.Available || s.Pending || !complete || f.calls != 1 || f.prepares != 1 {
		t.Fatal(s, err)
	}
	_, err = f.apply(s, nil)
	if err != nil || f.calls != 1 {
		t.Fatal("matching binary must not reinstall", err)
	}
}
func TestServiceUpgradeCancelledAndLostReplyRetainOriginalTarget(t *testing.T) {
	for _, lostReply := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "lost-reply"}[lostReply], func(t *testing.T) {
			f := newUpgradeFixture(t)
			s, _ := f.check()
			f.failure = errors.New("authorization cancelled or reply lost")
			if _, err := f.apply(s, nil); err == nil {
				t.Fatal("expected interruption")
			}
			if lostReply {
				f.installed = []byte("candidate")
			}
			if err := os.WriteFile(f.runner, []byte("newer desktop candidate"), 0700); err != nil {
				t.Fatal(err)
			}
			pending, err := f.check()
			if err != nil || !pending.Pending || pending.TargetSHA256 != s.TargetSHA256 {
				t.Fatal(pending, err)
			}
			f.failure = nil
			done, err := f.apply(pending, nil)
			if err != nil || done.Pending || !done.Available || f.prepares != 1 || string(f.installed) != "candidate" {
				t.Fatal(done, err)
			}
			if done.TargetSHA256 == s.TargetSHA256 {
				t.Fatal("next check should offer the newer desktop separately")
			}
		})
	}
}
func TestServiceUpgradeRefusesChangedApprovalAndBusyRunner(t *testing.T) {
	for _, change := range []string{"identity", "target", "busy"} {
		t.Run(change, func(t *testing.T) {
			f := newUpgradeFixture(t)
			s, _ := f.check()
			switch change {
			case "identity":
				s.RunnerID = "runner-other"
			case "target":
				s.TargetSHA256 = strings.Repeat("0", 64)
			case "busy":
				f.errorProbe = errors.New("pending ship work")
			}
			if _, err := f.apply(s, nil); err == nil || f.calls != 0 || f.prepares != 0 {
				t.Fatal("unsafe update", err)
			}
		})
	}
}
func TestServiceUpgradeRetryRefusesChangedRetainedInputs(t *testing.T) {
	for _, change := range []string{"binding", "binary", "key", "bundle", "version", "other-update"} {
		t.Run(change, func(t *testing.T) {
			f := newUpgradeFixture(t)
			s, _ := f.check()
			f.failure = errors.New("interrupted")
			_, _ = f.apply(s, nil)
			f.failure = nil
			switch change {
			case "other-update":
				f.installed = []byte("third-party generation")
			case "binding":
				f.binding.AccessSHA256 = "changed"
			case "binary":
				os.WriteFile(filepath.Join(f.dir, "service-upgrade", "urbit-runner"), []byte("changed"), 0700)
			case "key":
				os.WriteFile(filepath.Join(f.dir, "service-upgrade", "operator.pub"), []byte("changed"), 0600)
			case "bundle":
				f.deps.verifyBundle = func(string) error { return errors.New("changed bundle") }
			case "version":
				os.WriteFile(filepath.Join(f.dir, "service-upgrade.json"), []byte(`{"version":999}`), 0600)
			}
			if _, err := f.apply(s, nil); err == nil || f.calls != 1 {
				t.Fatal("retargeted update", err)
			}
		})
	}
}
func TestServiceUpgradeNoCompletionForWrongHostOrFailedReadiness(t *testing.T) {
	for _, change := range []string{"host", "access", "readiness"} {
		t.Run(change, func(t *testing.T) {
			f := newUpgradeFixture(t)
			s, _ := f.check()
			normal := f.deps.elevate
			f.deps.elevate = func(ctx context.Context, r, b, p, id string, emit func(Event)) (hostinstall.HostResult, error) {
				h, e := normal(ctx, r, b, p, id, emit)
				switch change {
				case "host":
					h.RunnerID = "runner-other"
				case "access":
					h.OperatorFingerprint = "other"
				case "readiness":
					f.errorProbe = errors.New("not ready")
				}
				return h, e
			}
			complete := false
			_, err := f.apply(s, func(e UpgradeEvent) { complete = complete || e.Step == "upgrade" && e.Status == "complete" })
			a, readErr := readUpgradeAttempt(f.dir)
			if err == nil || readErr != nil || a == nil || a.Complete || complete {
				t.Fatal("unverified success", err, readErr)
			}
		})
	}
}
func TestServiceUpgradeNewAttemptCannotGenerateAccess(t *testing.T) {
	f := newUpgradeFixture(t)
	s, _ := f.check()
	os.Remove(filepath.Join(f.dir, "operator.pub"))
	if _, err := f.apply(s, nil); err == nil || f.calls != 0 {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "operator.pub")); !os.IsNotExist(err) {
		t.Fatal("regenerated key")
	}
}
