package hostcompat

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestBootPredicateMatchesGoPolicy(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("POSIX shell unavailable")
	}
	for _, version := range []string{"27.1.999", "27.2.0", "28.0.0", "29.7.2", "29.9.1", "29.9.1+dfsg1", "29.9.1+build.01-foo", "29.9.1+", "29.9.1+a+b", "29.9.1+a..b", "29.9.1-rc1+build", "29.9.1+$(exit 0)", "29.999999.999999", "30.0.0", "29.9.1-rc1", "29.09.1", "029.9.1", "29.9.1000000", "29.9.1.", "29..1", "29.9.1\n", "29.9.1; exit 0", "$(exit 0)"} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		cmd := exec.CommandContext(ctx, "sh", "-c", EngineVersionShell()+"\nengine_version_supported \"$1\"", "version-test", version)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
		got := cmd.Run() == nil
		cancel()
		if want := EngineVersionStatus(version) == "compatible"; got != want {
			t.Fatalf("shell=%v, Go=%v for %q", got, want, version)
		}
	}
}
