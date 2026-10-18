package managed

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func receiptServer(t *testing.T, dir string) *Server {
	t.Helper()
	r, err := newServer(dir, &handoffBackend{dir: dir}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r.keys = filepath.Join(dir, "keys")
	t.Cleanup(func() { r.Close() })
	return r
}

func receiptActors(t *testing.T, dir string) (string, string, string) {
	t.Helper()
	line, actor := grantLine(t, "operator")
	otherLine, other := grantLine(t, "operator")
	observerLine, observer := grantLine(t, "observer")
	if err := os.WriteFile(filepath.Join(dir, "keys"), []byte(line+"\n"+otherLine+"\n"+observerLine), 0600); err != nil {
		t.Fatal(err)
	}
	return actor, other, observer
}

func receiptCall(t *testing.T, r *Server, actor, path string, body any, status int, problemCode ...string) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", path, bytes.NewReader(data))
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
	req.Header.Set(access.Header, actor)
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != status {
		t.Fatalf("%s: status=%d want=%d body=%s", path, w.Code, status, w.Body)
	}
	if status == 409 {
		code := "request_conflict"
		if len(problemCode) != 0 {
			code = problemCode[0]
		}
		if !strings.Contains(w.Body.String(), code) {
			t.Fatal("expected conflict", code, w.Body)
		}
	}
	return w
}

func receiptOffer(t *testing.T, r *Server, n int) (protocol.Handoff, ed25519.PrivateKey) {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Handoff{ID: fmt.Sprintf("%032x", n+1), Direction: "out", State: "ready", SourceRunner: "runner-source", SourceInstance: fmt.Sprintf("ship-%d", n), DestinationRunner: r.s.runnerID, DestinationInstance: fmt.Sprintf("arrival-%d", n), Ship: "~zod", Mode: "fake", SHA256: strings.Repeat("d", 64), TreeDigest: strings.Repeat("e", 64), Bytes: 1, VerifyKey: hex.EncodeToString(pub)}, key
}

func receiptSave(t *testing.T, r *Server, v storedInstance) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	tx, err := r.s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	current, err := loadInstance(tx, v.Public.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The initial poll runs immediately, even with a one-hour polling interval.
	// Model a runtime mutation's revision fence using the latest transactional
	// revision, not v's potentially stale pre-observation value. Otherwise an
	// in-flight poll can resurrect the transfer this fixture just replaced.
	v.RuntimeRevision = current.RuntimeRevision + 1
	if err = saveInstance(tx, v); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func receiptLoad(t *testing.T, r *Server, id string) storedInstance {
	t.Helper()
	v, err := loadInstance(r.s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func assertHandoffReceipt(t *testing.T, r *Server, actor, request, kind string, revision uint64, response *httptest.ResponseRecorder) {
	t.Helper()
	var raw []byte
	var fingerprint string
	if err := r.s.db.QueryRow("SELECT body,fingerprint FROM operations WHERE actor=? AND request_id=?", "ssh-key:"+actor, request).Scan(&raw, &fingerprint); err != nil {
		t.Fatal(err)
	}
	var stored handoffReceiptOperation
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	var h protocol.Handoff
	if err := json.Unmarshal(response.Body.Bytes(), &h); err != nil {
		t.Fatal(err)
	}
	if stored.Actor != "ssh-key:"+actor || stored.Kind != kind || stored.RequestID != request || stored.ConfigRevision != revision || stored.Status != "succeeded" || stored.FinishedAt == nil || stored.Receipt == nil || stored.Receipt.Status != response.Code || !reflect.DeepEqual(stored.Receipt.Handoff, h) || !strings.HasPrefix(fingerprint, "handoff-receipt-v1:") {
		t.Fatalf("invalid audit/replay receipt: %s", raw)
	}
}

func handoffAuditRow(t *testing.T, r *Server, actor, request string) string {
	t.Helper()
	var raw []byte
	var fingerprint string
	if err := r.s.db.QueryRow("SELECT body,fingerprint FROM operations WHERE actor=? AND request_id=?", actor, request).Scan(&raw, &fingerprint); err != nil {
		t.Fatal(err)
	}
	return string(raw) + "\x00" + fingerprint
}

func assertReceiptInstance(t *testing.T, got, want storedInstance) {
	t.Helper()
	// These fixtures stay fenced at reserved/received. A valid observation may
	// advance its bookkeeping revision and timestamp without changing any other
	// instance data. Still compare the entire runtime, signed handoff, public
	// configuration/intent, storage identity and all remaining fields exactly.
	got.RuntimeRevision, want.RuntimeRevision = 0, 0
	got.Public.ObservedAt, want.Public.ObservedAt = nil, nil
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		t.Fatalf("receipt changed authoritative instance state\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

// Drive the poller's persistence boundary in a fixed order, without depending on
// goroutine scheduling. These servers have a real store/HTTP handler but no poll
// loop; each s.observe call is exactly the write the production poller performs.
func TestHandoffReceiptObservationRevisionBinding(t *testing.T) {
	for _, boundary := range []string{"fixture-old-poll", "fixture-intervening-poll", "activation", "legacy-reservation"} {
		t.Run(boundary, func(t *testing.T) {
			s := testStore(t)
			actor, _, _ := receiptActors(t, s.directory)
			r := &Server{s: s, b: &handoffBackend{dir: s.directory}, uid: uint32(os.Geteuid()), keys: filepath.Join(s.directory, "keys")}
			h, key := receiptOffer(t, r, 0)
			request := protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: 18080, RequestID: "reserve"}
			receiptCall(t, r, actor, "/v1/handoffs", request, 201)
			audit := handoffAuditRow(t, r, "ssh-key:"+actor, request.RequestID)
			captured := receiptLoad(t, r, h.DestinationInstance)
			state, observation, err := r.b.Observe(context.Background(), captured.Runtime)
			if err != nil {
				t.Fatal(err)
			}
			// A separate load avoids pointer aliasing with the captured poll.
			v := receiptLoad(t, r, h.DestinationInstance)
			if strings.HasPrefix(boundary, "fixture-") {
				// One poll completes after the fixture read but before its write.
				// Incrementing the fixture's OLD revision would collide with this
				// new revision, so receiptSave must read the latest revision in tx.
				if err := s.observe(captured, state, observation); err != nil {
					t.Fatal(err)
				}
				if boundary == "fixture-intervening-poll" {
					captured = receiptLoad(t, r, h.DestinationInstance)
					state, observation, err = r.b.Observe(context.Background(), captured.Runtime)
					if err != nil {
						t.Fatal(err)
					}
				}
				v.Runtime.Handoff.ID = strings.Repeat("b", 32)
				v.Public.Handoff = v.Runtime.Handoff
				receiptSave(t, r, v)
			} else {
				v.Runtime.Handoff.State, v.Runtime.Initialized, v.Runtime.Prepared = "received", true, true
				v.Public.Handoff = v.Runtime.Handoff
				receiptSave(t, r, v)
				captured = receiptLoad(t, r, h.DestinationInstance)
				state, observation, err = r.b.Observe(context.Background(), captured.Runtime)
				if err != nil {
					t.Fatal(err)
				}
				if boundary == "activation" {
					finish := protocol.FinishHandoffRequest{RequestID: "activate", ExpectedRevision: 1, SHA256: h.SHA256, Commitment: hex.EncodeToString(ed25519.Sign(key, h.CommitmentBytes()))}
					receiptCall(t, r, actor, "/v1/handoffs/"+h.ID+"/activate", finish, 200)
				} else {
					// The legacy reservation accepts a new actor/request receipt
					// without updating runtime state. Its concurrent poll is valid.
					request.RequestID = "legacy-reserve"
					receiptCall(t, r, actor, "/v1/handoffs", request, 200)
				}
			}
			beforeObservation := receiptLoad(t, r, h.DestinationInstance)
			if err := s.observe(captured, state, observation); err != nil {
				t.Fatal(err)
			}
			after := receiptLoad(t, r, h.DestinationInstance)
			if boundary == "legacy-reservation" {
				if after.RuntimeRevision != beforeObservation.RuntimeRevision+1 || after.Public.ObservedAt == nil {
					t.Fatal("fresh poll was not accepted")
				}
				assertReceiptInstance(t, after, beforeObservation)
			} else if !reflect.DeepEqual(after, beforeObservation) {
				t.Fatalf("stale observation overwrote %s: captured revision=%d, saved revision=%d, after revision=%d, transfer=%s", boundary, captured.RuntimeRevision, beforeObservation.RuntimeRevision, after.RuntimeRevision, after.Runtime.Handoff.ID)
			}
			if got := handoffAuditRow(t, r, "ssh-key:"+actor, "reserve"); got != audit {
				t.Fatal("observation rewrote original audit receipt")
			}
		})
	}
}

func TestHandoffReservationReceiptsSurviveProgressAndRestart(t *testing.T) {
	dir := privateDir(t)
	actor, other, observer := receiptActors(t, dir)
	r := receiptServer(t, dir)
	h, signingKey := receiptOffer(t, r, 0)
	request := protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: 18080, RequestID: "reserve"}
	receiptCall(t, r, observer, "/v1/handoffs", request, 403)
	first := receiptCall(t, r, actor, "/v1/handoffs", request, 201)
	assertHandoffReceipt(t, r, actor, request.RequestID, "reserve-handoff", 1, first)
	changed := request
	changed.Offer, _ = receiptOffer(t, r, 1)
	changed.HTTPPort++
	receiptCall(t, r, actor, "/v1/handoffs", changed, 409)
	changed = request
	changed.HTTPPort = 0 // auto-selection is distinct input, even if it selects the same port
	receiptCall(t, r, actor, "/v1/handoffs", changed, 409)
	changed = request
	changed.Offer.Message = "different offer"
	receiptCall(t, r, actor, "/v1/handoffs", changed, 409)
	second := receiptCall(t, r, other, "/v1/handoffs", request, 200)
	assertHandoffReceipt(t, r, other, request.RequestID, "reserve-handoff", 1, second)
	audit := map[string]string{}
	for _, who := range []string{actor, other} {
		audit[who] = handoffAuditRow(t, r, "ssh-key:"+who, request.RequestID)
	}
	for _, phase := range []string{"reserved", "receiving", "received", "complete"} {
		v := receiptLoad(t, r, h.DestinationInstance)
		v.Runtime.Handoff.State = phase
		if phase == "receiving" {
			v.Runtime.Handoff.Error = "interrupted receipt"
		} else {
			v.Runtime.Handoff.Error = ""
		}
		v.Runtime.Initialized = phase == "received" || phase == "complete"
		if phase == "complete" {
			v.Runtime.Handoff.Commitment = hex.EncodeToString(ed25519.Sign(signingKey, h.CommitmentBytes()))
		}
		v.Public.Handoff = v.Runtime.Handoff
		receiptSave(t, r, v)
		r.Close()
		r = receiptServer(t, dir)
		for _, who := range []string{actor, other} {
			got := receiptCall(t, r, who, "/v1/handoffs", request, 200)
			var current protocol.Handoff
			if err := json.Unmarshal(got.Body.Bytes(), &current); err != nil || !reflect.DeepEqual(current, *v.Runtime.Handoff) {
				t.Fatal("reservation replay did not return current matching transfer", phase, got.Body, err)
			}
			if got := handoffAuditRow(t, r, "ssh-key:"+who, request.RequestID); got != audit[who] {
				t.Fatal("reservation replay rewrote its original audit receipt", phase)
			}
		}
		if got := receiptLoad(t, r, h.DestinationInstance); !reflect.DeepEqual(got.Runtime.Handoff, v.Runtime.Handoff) || got.Runtime.Initialized != v.Runtime.Initialized {
			t.Fatal("replay rolled back receipt progress")
		}
	}
	assertHandoffReceipt(t, r, actor, request.RequestID, "reserve-handoff", 1, first)
	assertHandoffReceipt(t, r, other, request.RequestID, "reserve-handoff", 1, second)
	changed.Offer, _ = receiptOffer(t, r, 2)
	receiptCall(t, r, actor, "/v1/handoffs", changed, 409)
}

func TestHandoffReservationReplayRefusesMissingOrReplacedTransfer(t *testing.T) {
	for _, change := range []string{"missing", "replacement", "changed-key", "direction"} {
		t.Run(change, func(t *testing.T) {
			dir := privateDir(t)
			actor, _, _ := receiptActors(t, dir)
			r := receiptServer(t, dir)
			h, _ := receiptOffer(t, r, 0)
			request := protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: 18080, RequestID: "reserve"}
			receiptCall(t, r, actor, "/v1/handoffs", request, 201)
			audit := handoffAuditRow(t, r, "ssh-key:"+actor, request.RequestID)
			v := receiptLoad(t, r, h.DestinationInstance)
			status := 404
			switch change {
			case "missing":
				v.Runtime.Handoff = nil
			case "replacement":
				v.Runtime.Handoff.ID = strings.Repeat("b", 32)
			case "changed-key":
				v.Runtime.Handoff.VerifyKey = strings.Repeat("b", 64)
				status = 409
			case "direction":
				v.Runtime.Handoff.Direction = "out"
				status = 409
			}
			v.Public.Handoff = v.Runtime.Handoff
			receiptSave(t, r, v)
			r.Close()
			r = receiptServer(t, dir)
			receiptCall(t, r, actor, "/v1/handoffs", request, status, "handoff_conflict")
			if got := handoffAuditRow(t, r, "ssh-key:"+actor, request.RequestID); got != audit {
				t.Fatal("unavailable transfer changed the accepted audit record")
			}
			if got := receiptLoad(t, r, h.DestinationInstance); !reflect.DeepEqual(got.Runtime.Handoff, v.Runtime.Handoff) {
				t.Fatal("replay replaced destination transfer")
			}
			request.HTTPPort++
			receiptCall(t, r, actor, "/v1/handoffs", request, 409)
		})
	}
}

func TestHandoffActivationReceiptsAndLegacySignatures(t *testing.T) {
	for _, settings := range []bool{false, true} {
		t.Run(fmt.Sprint(settings), func(t *testing.T) {
			dir := privateDir(t)
			actor, other, observer := receiptActors(t, dir)
			r := receiptServer(t, dir)
			h, key := receiptOffer(t, r, 0)
			if settings {
				h.RuntimeSettings = &protocol.RuntimeSettings{Loom: 29, NativeControl: true}
			}
			if _, err := r.s.db.Exec("INSERT INTO deleted_instances VALUES(?,?,?)", h.DestinationInstance, "old-delete", 7); err != nil {
				t.Fatal(err)
			}
			receive := protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: 18080, RequestID: "reserve"}
			receiptCall(t, r, actor, "/v1/handoffs", receive, 201)
			v := receiptLoad(t, r, h.DestinationInstance)
			v.Runtime.Handoff.State = "received"
			v.Runtime.Initialized, v.Runtime.Prepared = true, true
			v.Public.Handoff = v.Runtime.Handoff
			receiptSave(t, r, v)
			// Model a transfer accepted by a historical runner with no reserve or
			// activation receipts. Its original signing representation is retained.
			if _, err := r.s.db.Exec("DELETE FROM operations"); err != nil {
				t.Fatal(err)
			}
			legacy := receiptCall(t, r, actor, "/v1/handoffs", receive, 200)
			assertHandoffReceipt(t, r, actor, receive.RequestID, "reserve-handoff", 8, legacy)
			assertReceiptInstance(t, receiptLoad(t, r, h.DestinationInstance), v)
			path := "/v1/handoffs/" + h.ID + "/activate"
			finish := protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: 8, RequestID: "activate", Commitment: hex.EncodeToString(ed25519.Sign(key, h.CommitmentBytes()))}
			receiptCall(t, r, observer, path, finish, 403)
			collision := finish
			collision.RequestID = receive.RequestID
			receiptCall(t, r, actor, path, collision, 409)
			first := receiptCall(t, r, actor, path, finish, 200)
			assertHandoffReceipt(t, r, actor, finish.RequestID, "activate-handoff", 8, first)
			v = receiptLoad(t, r, h.DestinationInstance)
			if v.Runtime.Handoff.Fenced() || v.Public.DesiredState != "stopped" || !bytes.Equal(v.Runtime.Handoff.CommitmentBytes(), h.CommitmentBytes()) || !v.Runtime.Handoff.ValidCommitment(finish.Commitment) {
				t.Fatal("activation changed stopped intent or signed transfer", v)
			}
			independent := receiptCall(t, r, other, path, finish, 200)
			assertHandoffReceipt(t, r, other, finish.RequestID, "activate-handoff", 8, independent)
			changed := finish
			changed.ExpectedRevision++
			receiptCall(t, r, actor, path, changed, 409)
			changed = finish
			changed.Commitment = strings.Repeat("0", 128)
			receiptCall(t, r, actor, path, changed, 409)
			h2, key2 := receiptOffer(t, r, 1)
			receiptCall(t, r, actor, "/v1/handoffs", protocol.ReceiveHandoffRequest{Offer: h2, HTTPPort: 18081, RequestID: "reserve-two"}, 201)
			v2 := receiptLoad(t, r, h2.DestinationInstance)
			v2.Runtime.Handoff.State, v2.Runtime.Initialized = "received", true
			v2.Public.Handoff = v2.Runtime.Handoff
			receiptSave(t, r, v2)
			changed = protocol.FinishHandoffRequest{SHA256: h2.SHA256, ExpectedRevision: 1, RequestID: finish.RequestID, Commitment: hex.EncodeToString(ed25519.Sign(key2, h2.CommitmentBytes()))}
			receiptCall(t, r, actor, "/v1/handoffs/"+h2.ID+"/activate", changed, 409)
			if got := receiptLoad(t, r, h2.DestinationInstance); got.Runtime.Handoff.State != "received" {
				t.Fatal("conflicting request activated a second transfer")
			}
			// Later configuration and even disappearance of the live transfer do
			// not turn an acknowledged activation into a different replay result.
			v.Public.ConfigRevision++
			v.Runtime.Handoff, v.Public.Handoff = nil, nil
			receiptSave(t, r, v)
			r.Close()
			r = receiptServer(t, dir)
			got := receiptCall(t, r, actor, path, finish, 200)
			if got.Body.String() != first.Body.String() {
				t.Fatal("activation response changed after restart/progress", got.Body)
			}
			receiptCall(t, r, actor, "/v1/handoffs/"+h2.ID+"/activate", changed, 409)
		})
	}
}

func TestHandoffReceiptsRollbackWithState(t *testing.T) {
	for _, kind := range []string{"reserve", "activate"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			actor, _, _ := receiptActors(t, dir)
			r := receiptServer(t, dir)
			h, key := receiptOffer(t, r, 0)
			receive := protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: 18080, RequestID: "reserve"}
			var before storedInstance
			if kind == "activate" {
				receiptCall(t, r, actor, "/v1/handoffs", receive, 201)
				before = receiptLoad(t, r, h.DestinationInstance)
				before.Runtime.Initialized = true
				before.Runtime.Handoff.State = "received"
				before.Public.Handoff = before.Runtime.Handoff
				receiptSave(t, r, before)
			}
			// Fail after state mutation AND ordinary receipt insertion, at the
			// final private replay-body write. Neither may survive the transaction.
			if _, err := r.s.db.Exec(`CREATE TRIGGER fail_receipt BEFORE UPDATE OF body ON operations WHEN json_extract(NEW.body,'$.handoff_receipt') IS NOT NULL BEGIN SELECT RAISE(ABORT,'injected receipt failure'); END`); err != nil {
				t.Fatal(err)
			}
			if kind == "reserve" {
				receiptCall(t, r, actor, "/v1/handoffs", receive, 500)
			} else {
				finish := protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: 1, RequestID: "activate", Commitment: hex.EncodeToString(ed25519.Sign(key, h.CommitmentBytes()))}
				receiptCall(t, r, actor, "/v1/handoffs/"+h.ID+"/activate", finish, 500)
			}
			var count int
			if err := r.s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id=?", kind).Scan(&count); err != nil || count != 0 {
				t.Fatal("partial receipt committed", count, err)
			}
			r.Close()
			r = receiptServer(t, dir)
			if kind == "reserve" {
				if err := r.s.db.QueryRow("SELECT count(*) FROM instances").Scan(&count); err != nil || count != 0 {
					t.Fatal("reservation survived failed receipt", count, err)
				}
			} else {
				assertReceiptInstance(t, receiptLoad(t, r, h.DestinationInstance), before)
			}
		})
	}
}
