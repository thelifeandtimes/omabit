package rootless

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func TestGeneratedShellAndGoEnginePolicyAgree(t *testing.T) {
	for _, version := range []string{"27.1.99", "27.2.0", "27.99.1", "28.0.0", "29.7.2", "29.9.1", "29.999999.999999", "30.0.0", "26.9.9", "", "29.9", "29.9.1-rc1", "29.9.1+local", "029.9.1", "29.09.1", "29.9.01", "29.9.1000000", "29.9.1\n", "29.9.1;exit 0", "$(true)"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			// Execute only the pure generated version function, never the host guard.
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", hostcompat.EngineVersionShell()+"\nengine_version_supported \"$1\"", "policy-test", version)
			cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
			got := cmd.Run() == nil
			want := hostcompat.EngineVersionStatus(version) == "compatible"
			if got != want {
				t.Fatalf("shell=%t Go=%t for %q", got, want, version)
			}
		})
	}
}

func TestGuardSeparatesLauncherPinAndEngineCompatibility(t *testing.T) {
	files, err := ServiceFiles()
	if err != nil {
		t.Fatal(err)
	}
	check := string(files["check-engine.sh"])
	if strings.Contains(check, "Expected exactly Docker") || !strings.Contains(check, hostcompat.EngineVersionShell()) || !strings.Contains(check, LauncherSHA256) || !strings.Contains(check, InstallDir) {
		t.Fatal("guard lost independent policy or launcher pin")
	}
	unit := string(files[UnitName])
	if !strings.Contains(unit, "Environment=PATH="+SafePath) || !strings.Contains(unit, "--data-root=%h/.local/share/urbit-runner/docker") {
		t.Fatal("unit lost native path or data root")
	}
	if InstallDir != "/usr/local/lib/urbit-runner/rootless/29.7.2" {
		t.Fatal("historical launcher path migrated implicitly")
	}
	old, err := PreviousServiceFiles()
	if err != nil || !strings.Contains(string(old["check-engine.sh"]), "Expected exactly Docker version 29.7.2.") || !strings.Contains(string(old[UnitName]), "Environment=PATH=/usr/bin:/bin") {
		t.Fatal("previous generation not retained", err)
	}
}
