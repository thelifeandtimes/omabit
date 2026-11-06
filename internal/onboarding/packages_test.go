package onboarding

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
)

func TestMissingNativePackagesRemainOneClickSetup(t *testing.T) {
	d, dir, config, runner, calls := setupFixture(t)
	d.inspect = func() hostsetup.Report {
		return hostsetup.Report{Compatibility: &hostcompat.Report{Distribution: hostcompat.Distribution{ID: "omarchy", Provider: "pacman"}}, Checks: []hostsetup.Check{
			{ID: "platform", Status: "pass"}, {ID: "binary-dockerd", Status: "missing"}, {ID: "docker-version", Status: "unknown"},
			{ID: "binary-rootlesskit", Status: "missing"}, {ID: "binary-slirp4netns", Status: "missing"}, {ID: "binary-ssh-keygen", Status: "missing"},
		}}
	}
	s, err := inspect(context.Background(), config, dir, d)
	if err != nil || !s.CanSetup || s.Ready || *calls != 0 || !slices.Equal(s.Packages, []string{"docker", "openssh", "rootlesskit", "slirp4netns"}) {
		t.Fatal(s, err)
	}
	for _, c := range s.Checks {
		if c.Status == "missing" && !strings.Contains(c.Detail, strings.TrimPrefix(c.ID, "binary-")) {
			t.Fatal("missing prerequisite was anonymous", c)
		}
	}
	s, err = setup(context.Background(), config, dir, runner, d, nil)
	if err != nil || !s.Ready || *calls != 1 {
		t.Fatal("missing packages prevented the one authorized installation", s, err)
	}
}

func TestBundledDesktopKeyWorksWithoutOpenSSHAndPreservesExistingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "id_ed25519")
	pub, err := desktopKey(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if !bytes.HasPrefix(before, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Fatal("key is not in OpenSSH format")
	}
	// A real external implementation must be able to use the bundled key.
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Fatal("compatibility check requires ssh-keygen", err)
	}
	actual, err := exec.Command("ssh-keygen", "-y", "-P", "", "-f", path).Output()
	if err != nil || !slices.Equal(strings.Fields(string(pub))[:2], strings.Fields(string(actual))[:2]) {
		t.Fatal("OpenSSH cannot use the bundled key", err)
	}
	again, err := desktopKey(context.Background(), path)
	after, _ := os.ReadFile(path)
	if err != nil || !bytes.Equal(pub, again) || !bytes.Equal(before, after) {
		t.Fatal("retry rotated the desktop key", err)
	}
	legacy := filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", legacy).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	before, _ = os.ReadFile(legacy)
	if _, err := desktopKey(context.Background(), legacy); err != nil {
		t.Fatal("existing ssh-keygen key was rejected", err)
	}
	after, _ = os.ReadFile(legacy)
	if !bytes.Equal(before, after) {
		t.Fatal("existing key changed")
	}
}
