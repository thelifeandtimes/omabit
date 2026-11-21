package checkplan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func jsonValue(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNamedPlansPreserveLegacyPayloadAndCannotExecute(t *testing.T) {
	for _, fixture := range []struct {
		id     string
		legacy func() map[string]any
	}{{Lifecycle, LegacyRuntime}, {HTTP, LegacyHTTP}} {
		plan, err := Read(fixture.id)
		if err != nil {
			t.Fatal(err)
		}
		if plan.ExecutableHere || plan.Runtime.RuntimeQualified || plan.Version != 1 || !strings.Contains(plan.ExecutionStatus, "no shared test executor") {
			t.Fatal("plan acquired execution/qualification authority")
		}
		if !reflect.DeepEqual(jsonValue(t, plan.Runtime), jsonValue(t, fixture.legacy())) {
			t.Fatal("legacy payload differs", fixture.id)
		}
		digest := plan.SHA256
		plan.SHA256 = ""
		raw, err := json.Marshal(plan)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != digest {
			t.Fatal("plan digest does not bind complete descriptor")
		}
		plan.Runtime.Steps[0] = "changed"
		fresh, _ := Read(fixture.id)
		if fresh.Runtime.Steps[0] == "changed" {
			t.Fatal("shared mutable plan")
		}
	}
	if _, err := Read("run_shell"); err == nil {
		t.Fatal("unknown plan accepted")
	}
	plans, err := List()
	if err != nil || len(plans) != 2 {
		t.Fatal("unexpected plan inventory")
	}
}

// The opt-in predecessor is a real clean frozen12f78952 runner. Compare actual
// old command output, not a reconstructed old struct or changed schema marker.
func TestNamedPlansMatchActualFrozenLegacyRunner(t *testing.T) {
	path := os.Getenv("OMABIT_CHECKPLAN_OLD_RUNNER")
	if path == "" {
		t.Skip("explicit frozen predecessor metadata-only CLI required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != "20af5a66f00339d40800d997e9a24c29b7f1c21e9faed17716f5bb9f996d0feb" {
		t.Fatal("wrong frozen predecessor")
	}
	for _, fixture := range []struct{ id, command string }{{Lifecycle, "runtime-plan"}, {HTTP, "http-plan"}} {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		command := exec.CommandContext(ctx, path, fixture.command)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		out, err := command.Output()
		cancel()
		if err != nil || stderr.Len() != 0 {
			t.Fatal("predecessor plan failed", err)
		}
		var previous any
		decoder := json.NewDecoder(bytes.NewReader(out))
		decoder.UseNumber()
		if decoder.Decode(&previous) != nil {
			t.Fatal("invalid predecessor JSON")
		}
		plan, err := Read(fixture.id)
		if err != nil || !reflect.DeepEqual(previous, jsonValue(t, plan.Runtime)) {
			t.Fatal("actual legacy JSON changed", fixture.id, err)
		}
	}
}
