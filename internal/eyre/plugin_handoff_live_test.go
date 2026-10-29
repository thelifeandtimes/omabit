package eyre

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestApplicationPluginHandoffIsolatedLive(t *testing.T) {
	requireIsolatedAuthLab(t)
	if os.Getenv("OMABIT_PLUGIN_LAB") != "1" {
		t.Skip("enable the separate plugin artifacts in the lab harness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	owner, err := New("http://127.0.0.1:18081", "~zod")
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	credentials := make([][]byte, 0, 4)
	retain := func(c *Client) []byte {
		data, err := c.ExportSession()
		if err != nil {
			t.Fatal(err)
		}
		credentials = append(credentials, append([]byte(nil), data...))
		return data
	}
	// The test issuer retains cleanup ownership for every offered session. A
	// production sender needs the same journal before exposing the approval UI.
	defer func() {
		for _, data := range credentials {
			client, _ := New("http://127.0.0.1:18081", "~zod")
			if err := client.RestoreSession(data); err == nil {
				err = client.Logout(context.Background())
				if err != nil && !errors.Is(err, ErrAuth) {
					t.Error("owned session cleanup failed", err)
				}
			}
			client.Close()
		}
	}()
	ownerData := retain(owner)
	theme, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	themeData := retain(theme)
	dms, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dmsData := retain(dms)
	defer theme.Close()
	defer dms.Close()
	deliver := func(plugin, pluginID, requestID string, data []byte, expected string) {
		receiver := "/artifacts/plugins/" + plugin + "/application_connection.py"
		if plugin == "theme" {
			receiver = "/artifacts/plugins/theme/client/application_connection.py"
		}
		offer := map[string]any{"version": 1, "request_id": strings.Repeat(requestID, 32),
			"connection_id": strings.Repeat("c", 32), "plugin_id": pluginID,
			"profile": "auth-lab", "session": json.RawMessage(data)}
		body, err := json.Marshal(offer)
		if err != nil {
			t.Fatal("cannot encode private offer")
		}
		defer clear(body)
		command := exec.CommandContext(ctx, "/usr/bin/python3", "-B", receiver)
		command.Stdin = bytes.NewReader(body)
		out, runErr := command.Output() // private pipe; no cookie output is logged
		defer clear(out)
		var reply struct {
			Version      int    `json:"version"`
			Status       string `json:"status"`
			RequestID    string `json:"request_id"`
			ConnectionID string `json:"connection_id"`
			PluginID     string `json:"plugin_id"`
			Profile      string `json:"profile"`
		}
		if len(out) > 4096 || json.Unmarshal(out, &reply) != nil || reply.Version != 1 || reply.Status != expected {
			t.Fatal("receiver did not return the expected bounded acknowledgement", plugin)
		}
		if expected == "accepted" && (runErr != nil || reply.RequestID != offer["request_id"] || reply.ConnectionID != offer["connection_id"] || reply.PluginID != pluginID || reply.Profile != "auth-lab") {
			t.Fatal("receiver acknowledgement binding failed", plugin)
		}
		if expected == "rejected" && runErr == nil {
			t.Fatal("receiver rejection returned success")
		}
	}
	deliver("theme", "thelifeandtimes.urbit-theme.auth-lab", "1", themeData, "accepted")
	deliver("dms", "urbit.dms.auth-lab", "2", dmsData, "accepted")
	// A duplicate must not replace the original plugin credential. Its new
	// session remains issuer-owned and is revoked on refusal.
	duplicate, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	duplicateData := retain(duplicate)
	deliver("theme", "thelifeandtimes.urbit-theme.auth-lab", "3", duplicateData, "rejected")
	if err := duplicate.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	duplicate.Close()
	owner.Close()
	theme.Close()
	dms.Close()
	probe := func(plugin, mode string) {
		command := exec.CommandContext(ctx, "/usr/bin/python3", "-B", "/artifacts/plugin_probe.py", plugin, mode)
		if mode == "standalone" {
			input, _ := json.Marshal(map[string]string{"code": fakeCode})
			defer clear(input)
			command.Stdin = bytes.NewReader(input)
		}
		out, err := command.Output()
		defer clear(out)
		var result struct {
			OK bool `json:"ok"`
		}
		if err != nil || len(out) > 4096 || json.Unmarshal(out, &result) != nil || !result.OK {
			t.Fatal("direct plugin probe failed", plugin, mode)
		}
	}
	probe("theme", "read")
	probe("dms", "read")
	probe("dms", "logout")
	check := func(data []byte, want error) {
		client, _ := New("http://127.0.0.1:18081", "~zod")
		defer client.Close()
		if err := client.RestoreSession(append([]byte(nil), data...)); err != nil {
			t.Fatal(err)
		}
		if err := client.Check(ctx); !errors.Is(err, want) {
			t.Fatal("unexpected independent-session state", err)
		}
	}
	check(dmsData, ErrAuth)
	check(themeData, nil)
	check(ownerData, nil)
	probe("theme", "read")
	probe("theme", "logout")
	check(themeData, ErrAuth)
	check(ownerData, nil)
	for _, plugin := range []string{"theme", "dms"} {
		probe(plugin, "standalone")
		probe(plugin, "logout")
	}
	t.Log("PASS: real Python receivers and isolated Secret Service; direct plugin reads/SSE after issuer closure; duplicate refusal; independent logout; standalone login without Omabit")
}
