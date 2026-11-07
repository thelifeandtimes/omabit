//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskReceiverAdapterKeepsTypedSelectionAndNoRetry(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
	if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
		t.Fatal(err)
	}
	response := `{"present":false,"ship":"","desk":"","zest":"","hash":""}`
	effects := 0
	lost := false
	f.d.executeInput = func(_ context.Context, args, env []string, body []byte) (string, error) {
		if strings.Contains(strings.Join(args, " "), "kiln-install") {
			t.Fatal("request in argv")
		}
		var req struct {
			Source map[string]json.RawMessage `json:"source"`
			Sink   map[string]json.RawMessage `json:"sink"`
		}
		if json.Unmarshal(body, &req) != nil {
			t.Fatal("invalid request")
		}
		if string(req.Source["dojo"]) == `"our"` {
			return `"~zod"`, nil
		}
		if string(req.Sink["app"]) == `"hood"` {
			effects++
			if !strings.Contains(string(body), "kiln-install") || !strings.Contains(string(body), "[%example ~nec %published]") {
				t.Fatal("publisher changed", string(body))
			}
			if lost {
				return "PRIVATE", errors.New("PRIVATE lost response")
			}
			return `">="`, nil
		}
		b := []byte(response)
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}
		out, _ := json.Marshal("0x" + hex.EncodeToString(b))
		return string(out), nil
	}
	runtime, err := openManagedDesk(context.Background(), state, id, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if sub, err := runtime.Subscription(context.Background(), "example"); err != nil || sub.Present {
		t.Fatal(sub, err)
	}
	response = `{"present":false,"ship":"~nec","desk":"","zest":"","hash":""}`
	if _, err = runtime.Subscription(context.Background(), "example"); err == nil {
		t.Fatal("contradictory subscription accepted")
	}
	for _, value := range []string{"false", "true", `"not a boolean"`} {
		response = value
		ready, err := runtime.ReceiverSourceReady(context.Background(), "example")
		if value == `"not a boolean"` {
			if err == nil {
				t.Fatal("malformed arrival state accepted")
			}
		} else if err != nil || ready != (value == "true") {
			t.Fatal("arrival observation", ready, err)
		}
	}
	p := deskwork.Publisher{Ship: "~nec", Desk: "published", ExpectedHash: "0v123"}
	if err = runtime.InstallRemote(context.Background(), "example", p); err != nil || effects != 1 {
		t.Fatal(err, effects)
	}
	lost = true
	if err = runtime.InstallRemote(context.Background(), "example", p); err == nil || strings.Contains(err.Error(), "PRIVATE") || effects != 2 {
		t.Fatal("lost reply leaked or retried", err, effects)
	}
}
