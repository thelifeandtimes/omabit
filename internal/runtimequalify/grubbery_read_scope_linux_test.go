//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestGrubberyReadScopeAuthenticatesOnceAndRequiresCleanupAndExactLifetime(t *testing.T) {
	for _, scenario := range []string{"ok", "callback-error", "logout-failed", "foreign-binding", "context-cancelled", "process-changed", "framework-changed"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(t.Context(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			code := "lidlut-tabwed-pillex-ridrup"
			cookie := "urbauth-~zod=scope-private-token"
			counts := map[string]int{}
			baseline := deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}
			f.d.executeInput = func(ctx context.Context, args, env []string, input []byte) (string, error) {
				if ctx.Err() != nil {
					return "", ctx.Err()
				}
				for _, arg := range args {
					if strings.Contains(arg, code) || strings.Contains(arg, cookie) {
						t.Fatal("credential entered argv")
					}
				}
				var lens struct {
					Source struct {
						Dojo string `json:"dojo"`
					} `json:"source"`
				}
				if json.Unmarshal(input, &lens) == nil && lens.Source.Dojo != "" {
					value := "~zod\n"
					if lens.Source.Dojo == "+code" {
						counts["code"]++
						value = code
					} else if lens.Source.Dojo != "our" {
						data, _ := json.Marshal(baseline)
						for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
							data[i], data[j] = data[j], data[i]
						}
						value = "0x" + hex.EncodeToString(data)
					}
					data, _ := json.Marshal(value)
					return string(data), nil
				}
				if len(args) < 2 || !strings.Contains(strings.Join(args, " "), "container exec --interactive "+id) {
					t.Fatal("unbound native command")
				}
				method, rawURL := args[len(args)-2], args[len(args)-1]
				u, e := url.Parse(rawURL)
				if e != nil || u.Scheme+"://"+u.Host != "http://127.0.0.1:8080" {
					t.Fatal("unbound app origin")
				}
				parts := bytes.SplitN(input, []byte("\n"), 3)
				if len(parts) != 3 {
					t.Fatal("invalid separate stdin fields")
				}
				counts[u.Path]++
				status, body, headers := 200, "~zod", ""
				switch u.Path {
				case "/~/host":
					if method != "GET" {
						t.Fatal("identity effect")
					}
				case "/~/name":
					if method != "GET" || string(parts[1]) != cookie {
						t.Fatal("missing exact owner identity cookie")
					}
				case "/~/login":
					if method != "POST" || string(parts[2]) != "password="+code {
						t.Fatal("login payload changed")
					}
					headers, body = "Set-Cookie: "+cookie+"; Path=/; Max-Age=600\r\n", ""
				case "/~/logout":
					if method != "POST" || string(parts[1]) != cookie {
						t.Fatal("logout lost exact owner session")
					}
					status, body = 204, ""
					if scenario == "logout-failed" {
						status = 500
					}
				default:
					if method != "GET" || !strings.HasPrefix(u.Path, "/grubbery/api/peek/") || string(parts[1]) != cookie || len(parts[2]) != 0 {
						t.Fatal("scope attempted unselected read or write", method, u.Path)
					}
					counts["app-reads"]++
					status, body = 404, "Not found"
				}
				return fmt.Sprintf("HTTP/1.1 %d response\r\nContent-Type: text/plain\r\n%sContent-Length: %d\r\n\r\n%s", status, headers, len(body), body), nil
			}
			runtime, err := openManagedDesk(t.Context(), state, id, f.d)
			if err != nil {
				t.Fatal(err)
			}
			continuity, err := runtime.SourceIdentity(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			binding := grubbery.Binding{RunnerID: "runner", Mode: "simulation", Actor: "operator", Target: apppolicy.Target{InstanceID: "alpha", Desk: "grubbery"}, RuntimeID: id, Continuity: continuity, FrameworkReceiptID: strings.Repeat("d", 64), ConfigRevision: 1, FrameworkBaseline: baseline, AppID: "state"}
			if err = binding.Validate(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var retained grubbery.NativeReader
			callbackErr := errors.New("bounded callback refusal")
			cleanup, err := runtime.WithGrubberyReads(ctx, binding, func(reader grubbery.NativeReader) error {
				retained = reader
				selected := binding
				if scenario == "foreign-binding" {
					selected.Actor = "other"
				}
				if scenario == "context-cancelled" {
					cancel()
				}
				for _, purpose := range []grubbery.ReadPurpose{grubbery.ReadSourceRoot, grubbery.ReadDeskRoot, grubbery.ReadSourceConfig} {
					request, e := selected.Read(purpose)
					if e != nil {
						return e
					}
					response, e := reader.GrubberyRead(context.Background(), request)
					if e != nil {
						return e
					}
					if response.Status != 404 || response.SessionCleanup != grubbery.NativeReadScopeActive {
						t.Fatal("provisional read became final evidence")
					}
					clear(response.Body)
				}
				if scenario == "callback-error" {
					return callbackErr
				}
				if scenario == "process-changed" {
					for _, container := range f.containers {
						mapAt(container.doc, "State")["Pid"] = 99999
					}
				}
				if scenario == "framework-changed" {
					baseline.Revision++
				}
				return nil
			})
			if scenario == "ok" && (err != nil || cleanup != "confirmed" || counts["app-reads"] != 3) {
				t.Fatal("bounded read scope did not finish", cleanup, err, counts)
			}
			if scenario != "ok" && err == nil {
				t.Fatal("failed scope was accepted", scenario, cleanup)
			}
			if scenario == "logout-failed" && cleanup != "unconfirmed" || scenario == "callback-error" && (!errors.Is(err, callbackErr) || cleanup != "confirmed") {
				t.Fatal("callback/cleanup outcomes were conflated", cleanup, err)
			}
			if counts["code"] != 1 || counts["/~/login"] != 1 || scenario != "process-changed" && counts["/~/logout"] != 1 {
				t.Fatal("scope repeated authentication or omitted cleanup", counts)
			}
			if (scenario == "foreign-binding" || scenario == "context-cancelled") && counts["app-reads"] != 0 {
				t.Fatal("foreign/expired scope read native data")
			}
			before := make(map[string]int, len(counts))
			for key, count := range counts {
				before[key] = count
			}
			request, _ := binding.Read(grubbery.ReadSourceRoot)
			if _, err = retained.GrubberyRead(context.Background(), request); err == nil || !reflect.DeepEqual(before, counts) {
				t.Fatal("retained reader escaped its lifetime")
			}
		})
	}
}
