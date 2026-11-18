package eyre

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func nativeAppPlan(t *testing.T) (grubbery.Plan, []deskwork.SourceFile) {
	t.Helper()
	files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: []byte(":: constructor fixture; not compilation evidence\n")}, {Path: "assets/raw.bin", Bytes: []byte{0, 255, 1, 0}}}
	entries := []deskwork.File{}
	for _, f := range files {
		entries = append(entries, deskwork.File{Path: f.Path, SHA256: sampleResponseHash(f.Bytes), Bytes: int64(len(f.Bytes))})
	}
	m, err := deskwork.NewManifest(entries)
	if err != nil {
		t.Fatal(err)
	}
	b := grubbery.Binding{RunnerID: "runner", Mode: "managed-fake", Actor: "local-uid:1000", Target: apppolicy.Target{InstanceID: "alpha", Desk: "grubbery"}, RuntimeID: strings.Repeat("a", 64), Continuity: strings.Repeat("b", 64), FrameworkReceiptID: strings.Repeat("c", 64), ConfigRevision: 1, FrameworkBaseline: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, AppID: "state"}
	p, err := grubbery.NewPlan(b, strings.Repeat("d", 64), strings.Repeat("e", 64), m, files, grubbery.Before{SnapshotDigest: strings.Repeat("f", 64), SourceState: "absent", DeskState: "absent"})
	if err != nil {
		t.Fatal(err)
	}
	return p, files
}
func sampleResponseHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestTypedGrubberyCallsPreserveExactRequestAndDoNotRetry(t *testing.T) {
	p, files := nativeAppPlan(t)
	var selected grubbery.Step
	for _, s := range p.Steps {
		if s.SourcePath == "assets/raw.bin" {
			selected = s
			break
		}
	}
	expected, err := p.Request(selected.ID, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"ok", "lost-reply", "redirect", "409", "400", "401", "403", "404", "500", "wrong-ack", "oversized", "wrong-identity", "expired", "changed-source"} {
		t.Run(variant, func(t *testing.T) {
			effectCalls := 0
			checks := 0
			transport := docketTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Scheme+"://"+r.URL.Host != "http://127.0.0.1:18081" || r.GetBody != nil {
					t.Fatal("origin changed or request replayable")
				}
				status, body := 200, "~zod"
				header := make(http.Header)
				switch r.URL.Path {
				case "/~/host":
					checks++
					if r.Header.Get("Cookie") != "" {
						t.Fatal("probe carried cookie")
					}
					if variant == "wrong-identity" {
						body = "~nec"
					}
				case "/~/name":
					checks++
					if r.Header.Get("Cookie") != "urbauth-~zod=private-fixture-cookie" {
						t.Fatal("auth missing")
					}
				default:
					effectCalls++
					if r.Method != expected.Method || r.URL.Path != expected.Path || r.URL.RawQuery != expected.Query || r.Header.Get("Content-Type") != expected.ContentType || r.Header.Get("Cookie") != "urbauth-~zod=private-fixture-cookie" {
						t.Fatal("original phase request changed")
					}
					got, err := io.ReadAll(r.Body)
					if err != nil || !bytes.Equal(got, expected.Body) {
						t.Fatal("binary body changed")
					}
					status, body = 201, "Created"
					switch variant {
					case "lost-reply":
						return nil, errors.New("PRIVATE transport detail")
					case "redirect":
						status = 307
						header.Set("Location", "http://other.invalid/steal")
					case "400":
						status = 400
						body = "PRIVATE refusal"
					case "401":
						status = 401
						body = "PRIVATE refusal"
					case "403":
						status = 403
						body = "PRIVATE refusal"
					case "404":
						status = 404
						body = "PRIVATE refusal"
					case "409":
						status = 409
						body = "PRIVATE conflict"
					case "500":
						status = 500
						body = "PRIVATE possible partial effect"
					case "wrong-ack":
						body = "not the pinned acknowledgement"
					case "oversized":
						body = strings.Repeat("x", maxBody+1)
					}
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			c, err := NewWithTransport("http://127.0.0.1:18081", "~zod", transport)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			c.session = session{Origin: c.origin, Ship: c.ship, Name: "urbauth-~zod", Value: "private-fixture-cookie", Expires: time.Now().Add(time.Hour)}
			local := append([]deskwork.SourceFile{}, files...)
			if variant == "expired" {
				c.session.Expires = time.Now().Add(-time.Hour)
			}
			if variant == "changed-source" {
				local[0].Bytes = []byte(`{"foreign":"/state"}`)
			}
			result, err := c.GrubberyStep(context.Background(), p, selected.ID, local)
			wantSuccess := variant == "ok"
			if wantSuccess != (err == nil) || effectCalls > 1 || err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unsafe native result", err, effectCalls)
			}
			known := variant == "400" || variant == "401" || variant == "403" || variant == "404" || variant == "409"
			if known != errors.Is(err, grubbery.ErrNativeRejected) {
				t.Fatal("known refusal/uncertainty conflated", variant, err)
			}
			if variant == "changed-source" && checks != 0 {
				t.Fatal("changed source contacted runtime")
			}
			if (variant == "wrong-identity" || variant == "expired" || variant == "changed-source") && effectCalls != 0 {
				t.Fatal("failed preflight submitted effect")
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "PRIVATE") || strings.Contains(string(encoded), "cookie") {
				t.Fatal("private body/cookie entered JSON result")
			}
		})
	}
}
func TestBoundGrubberyReadsWorkWithoutSourceCacheAndPreserveAbsence(t *testing.T) {
	p, _ := nativeAppPlan(t)
	read, err := p.Binding.Read(grubbery.ReadSourceOwner)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := read.Request()
	calls := 0
	transport := docketTransport(func(r *http.Request) (*http.Response, error) {
		status, body := 200, "~zod"
		if strings.HasPrefix(r.URL.Path, "/grubbery/") {
			calls++
			if r.Method != "GET" || r.URL.Path != expected.Path || r.URL.RawQuery != "" {
				t.Fatal("read changed authority")
			}
			status, body = 404, "Not found"
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	c, err := NewWithTransport("http://127.0.0.1:18081", "~zod", transport)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.session = session{Origin: c.origin, Ship: c.ship, Name: "urbauth-~zod", Value: "private", Expires: time.Now().Add(time.Hour)}
	got, err := c.GrubberyRead(context.Background(), read)
	if err != nil || got.Status != 404 || calls != 1 || string(got.Body) != "Not found" {
		t.Fatal("absence lost", got.Status, err)
	}
	if _, err = c.GrubberyRead(context.Background(), grubbery.ReadRequest{}); err == nil || calls != 1 {
		t.Fatal("zero read contacted runtime")
	}
}
