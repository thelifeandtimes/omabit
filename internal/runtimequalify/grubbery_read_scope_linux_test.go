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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestGrubberyReadScopeAuthenticatesOnceAndRequiresCleanupAndExactLifetime(t *testing.T) {
	for _, scenario := range []string{"ok", "callback-error", "logout-failed", "foreign-binding", "context-cancelled", "process-changed", "framework-changed", "recovery-view", "recovery-context-changed", "recovery-saved"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			f.engine.ID = "fixture-recovery-daemon"
			state, _, err := applyManaged(t.Context(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			for _, c := range f.containers {
				mapAt(c.doc, "State")["Pid"] = 4242
				mapAt(c.doc, "State")["StartedAt"] = "2026-01-01T00:00:00Z"
			}
			namespace := "4:1001"
			f.d.readBootID = func() (string, error) { return "11111111-2222-3333-4444-555555555555", nil }
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) >= 4 && reflect.DeepEqual(args[4:], []string{"container", "exec", id, "/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
					return namespace, nil
				}
				return execute(ctx, args, env)
			}
			if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			code := "lidlut-tabwed-pillex-ridrup"
			cookie := "urbauth-~zod=scope-private-token"
			counts := map[string]int{}
			var expectedEffect *grubbery.Request
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
					if expectedEffect != nil && method != "GET" {
						if method != expectedEffect.Method || u.Path != expectedEffect.Path || u.RawQuery != expectedEffect.Query || string(parts[0]) != expectedEffect.ContentType || string(parts[1]) != cookie || !bytes.Equal(parts[2], expectedEffect.Body) {
							t.Fatal("recovered request changed original payload/selection")
						}
						counts["app-writes"]++
						status, body = 201, "Created"
						break
					}
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
			var recoveredPlan grubbery.Plan
			var recoveredFiles []deskwork.SourceFile
			if strings.HasPrefix(scenario, "recovery-") {
				binding.RuntimeID = strings.Repeat("9", 64)
				if runtime.grubberyBinding(t.Context(), binding) == nil {
					t.Fatal("ordinary runtime accepted an old container binding")
				}
				process, e := runtime.GrubberyProcess(t.Context())
				if e != nil {
					t.Fatal(e)
				}
				if scenario == "recovery-saved" {
					record, receipt, files := recoveryAdapterReceipt(t, binding, process)
					recoveredPlan, recoveredFiles = record.Plan, files
					invalid := receipt
					invalid.Review.Process.PID++
					if _, e = runtime.GrubberyRecovered(t.Context(), record, invalid); e == nil {
						t.Fatal("changed receipt body accepted")
					}
					runtime, e = runtime.GrubberyRecovered(t.Context(), record, receipt)
				} else {
					runtime, e = runtime.GrubberyRecoveryView(t.Context(), binding, process)
				}
				if e != nil {
					t.Fatal(e)
				}
				before := make(map[string]int)
				for key, count := range counts {
					before[key] = count
				}
				if scenario != "recovery-saved" {
					if _, e = runtime.GrubberyStep(t.Context(), grubbery.Plan{}, "", nil); e == nil || !reflect.DeepEqual(before, counts) {
						t.Fatal("review view exposed a native write")
					}
				}
				currentBinding := binding
				currentBinding.RuntimeID = id
				if runtime.grubberyBinding(t.Context(), currentBinding) == nil {
					t.Fatal("recovery view widened its original binding")
				}
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
				if scenario == "recovery-context-changed" {
					namespace = "4:1002"
				}
				return nil
			})
			success := scenario == "ok" || scenario == "recovery-view" || scenario == "recovery-saved"
			if success && (err != nil || cleanup != "confirmed" || counts["app-reads"] != 3) {
				t.Fatal("bounded read scope did not finish", cleanup, err, counts)
			}
			if !success && err == nil {
				t.Fatal("failed scope was accepted", scenario, cleanup)
			}
			if scenario == "logout-failed" && cleanup != "unconfirmed" || scenario == "callback-error" && (!errors.Is(err, callbackErr) || cleanup != "confirmed") {
				t.Fatal("callback/cleanup outcomes were conflated", cleanup, err)
			}
			if counts["code"] != 1 || counts["/~/login"] != 1 || scenario != "process-changed" && scenario != "recovery-context-changed" && counts["/~/logout"] != 1 {
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
			if scenario == "recovery-saved" {
				// Adapter-level inert transport check only; the runner must still
				// hold admission and commit the original dispatch before entry.
				original, _ := json.Marshal(recoveredPlan)
				step := recoveredPlan.Steps[0].ID
				selected, e := recoveredPlan.Request(step, recoveredFiles)
				if e != nil {
					t.Fatal(e)
				}
				expectedEffect = &selected
				response, e := runtime.GrubberyStep(t.Context(), recoveredPlan, step, recoveredFiles)
				if e != nil || response.SessionCleanup != "confirmed" || counts["app-writes"] != 1 {
					t.Fatal("original exact request did not reach selected replacement once", response, e, counts)
				}
				after, _ := json.Marshal(recoveredPlan)
				if !bytes.Equal(original, after) || recoveredPlan.Binding.RuntimeID == id {
					t.Fatal("adapter rewrote original plan to select replacement")
				}
				namespace = "4:1002"
				if _, e = runtime.GrubberyStep(t.Context(), recoveredPlan, step, recoveredFiles); e == nil || counts["app-writes"] != 1 {
					t.Fatal("changed approved namespace dispatched another effect")
				}
			}
		})
	}
}

func recoveryAdapterReceipt(t *testing.T, binding grubbery.Binding, process grubbery.RuntimeProcess) (grubbery.Record, grubbery.RuntimeRecoveryReceipt, []deskwork.SourceFile) {
	t.Helper()
	sample, err := appdev.ReadSample("grubbery-state-v1")
	if err != nil {
		t.Fatal(err)
	}
	files := []deskwork.SourceFile{}
	for _, file := range sample.Files {
		files = append(files, deskwork.SourceFile{Path: file.Path, Bytes: []byte(file.Content)})
	}
	plan, err := grubbery.NewPlan(binding, strings.Repeat("d", 64), strings.Repeat("1", 64), sample.Source, files, grubbery.Before{SnapshotDigest: strings.Repeat("6", 64), SourceState: "absent", DeskState: "absent"})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	record, err := grubbery.NewRecord(plan, at.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	review := grubbery.RuntimeRecoveryReview{Version: 1, OriginalRequestID: plan.RequestID, PlanDigest: plan.Digest, HeaderDigest: record.HeaderDigest, OriginalEventDigest: record.HeaderDigest, StepID: plan.Steps[0].ID, State: "pending", OriginalBinding: binding, PreviousContainerID: binding.RuntimeID, Process: process, EvidenceBasis: "verified-pending-prefix", NativeEvidenceDigest: strings.Repeat("6", 64), ObservedAt: at}
	review.Digest = review.ContentDigest()
	receipt := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: strings.Repeat("5", 64), Review: review, AppliedAt: at.Add(time.Second)}
	receipt.Digest = receipt.ContentDigest()
	if err = receipt.ValidateAncestry(record); err != nil {
		t.Fatal(err)
	}
	return record, receipt, files
}
