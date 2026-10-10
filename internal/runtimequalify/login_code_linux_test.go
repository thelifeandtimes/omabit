//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLoginCodeUsesFixedPrivateHTTPAndRedactsFailures(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
	execute := f.d.execute
	calls := 0
	response := `"lidlut-tabwed-pillex-ridrup\n"`
	var failure error
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "http://127.0.0.1:12321") {
			calls++
			joined := strings.Join(args, " ")
			for _, required := range []string{"container exec " + id, "--disable", "--noproxy *", "--max-filesize 1024", `{"source":{"dojo":"+code"},"sink":{"stdout":null}}`} {
				if !strings.Contains(joined, required) {
					t.Fatal("unsafe code command", required)
				}
			}
			if strings.Contains(joined, "conn.sock") || strings.Contains(joined, "lidlut") {
				t.Fatal("secret or control socket in command")
			}
			return response, failure
		}
		return execute(ctx, args, env)
	}
	code, err := readLoginCode(context.Background(), state, id, f.d)
	if err != nil || code != "lidlut-tabwed-pillex-ridrup" || calls != 1 {
		t.Fatal("code retrieval failed", err)
	}
	for _, out := range []string{`{"code":"PRIVATE"}`, `"PRIVATE"`, strings.Repeat("PRIVATE", 200)} {
		response = out
		code, err = readLoginCode(context.Background(), state, id, f.d)
		if err == nil || code != "" || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal("bad output accepted or leaked")
		}
	}
	response = "PRIVATE"
	failure = errors.New("PRIVATE")
	if code, err = readLoginCode(context.Background(), state, id, f.d); err == nil || code != "" || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatal("failed command output leaked")
	}
	before := calls
	if _, err := readLoginCode(context.Background(), state, strings.Repeat("f", 64), f.d); err == nil {
		t.Fatal("wrong container accepted")
	}
	state.MaintenanceUncertain = true
	if _, err := readLoginCode(context.Background(), state, id, f.d); err == nil || calls != before {
		t.Fatal("quarantined runtime queried")
	}
}
