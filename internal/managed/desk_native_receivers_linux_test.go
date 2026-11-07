//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (f *nativeDeskFixture) Subscription(ctx context.Context, desk string) (deskwork.Subscription, error) {
	var out deskwork.Subscription
	q, err := deskwork.SubscriptionInventory(f.ship(), desk)
	err = f.decode(ctx, q, err, &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}
func (f *nativeDeskFixture) InstallRemote(ctx context.Context, desk string, p deskwork.Publisher) error {
	q, err := deskwork.InstallRemote(f.ship(), desk, p)
	err = f.effect(ctx, q, err)
	if err == nil && f.loseReply.Swap(false) {
		return errors.New("qualification: discard real receiver acknowledgement")
	}
	return err
}
func (f *nativeDeskFixture) ReceiverSourceReady(ctx context.Context, desk string) (bool, error) {
	var ready bool
	q, err := deskwork.ReceiverSourceReady(f.ship(), desk)
	err = f.decode(ctx, q, err, &ready)
	return ready, err
}

// This function reads only the private fake's login code. Neither code, cookies,
// authentication request bodies nor responses are written to evidence logs.
func nativeReceiverFrontend(ctx context.Context, f *nativeDeskFixture, content []byte) error {
	code, err := f.post(ctx, []byte(`{"source":{"dojo":"+code"},"sink":{"stdout":null}}`))
	if err != nil {
		return errors.New("private fake code unavailable")
	}
	password := []byte(strings.TrimPrefix(strings.TrimSpace(code), "~"))
	code = ""
	owner, err := eyre.New(fmt.Sprintf("http://127.0.0.1:%d", f.HTTPPort), f.ship())
	if err != nil {
		clear(password)
		return err
	}
	defer owner.Close()
	err = owner.Login(ctx, password)
	clear(password)
	if err != nil {
		return errors.New("private fake login unavailable")
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = owner.Logout(c)
	}()
	secret, err := owner.ExportSession()
	if err != nil {
		return err
	}
	var session struct {
		Name  string `json:"cookie_name"`
		Value string `json:"cookie_value"`
	}
	err = json.Unmarshal(secret, &session)
	clear(secret)
	if err != nil {
		return errors.New("private frontend session unavailable")
	}
	for end := time.Now().Add(40 * time.Second); ; {
		if err = f.check(); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://127.0.0.1:%d/apps/%s/index.html", f.HTTPPort, f.Desk), nil)
		if err != nil {
			return err
		}
		req.AddCookie(&http.Cookie{Name: session.Name, Value: session.Value})
		response, err := f.client.Do(req)
		if err == nil {
			body, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			response.Body.Close()
			if e == nil && response.StatusCode == 200 && bytes.Equal(body, content) {
				return f.check()
			}
		}
		if time.Now().After(end) || ctx.Err() != nil {
			return errors.New("receiver frontend did not serve exact frozen HTML")
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// The optional launcher starts two fresh fake peers in the same isolated
// loopback-only namespace. Lifecycle remains synthetic; receiver API, collector,
// policy, journal and typed native ship effects are exercised together.
func TestNativeDeskReceivers(t *testing.T) {
	if _, err := os.Stat("/lab/receivers-runtime.json"); os.IsNotExist(err) {
		t.Skip("explicit private receiver capsule required")
	}
	publisher := nativeDeskGuard(t)
	publisher.ledger = "receiver-zod-effects.jsonl"
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	data, err := os.ReadFile("/lab/receivers-runtime.json")
	g02Check(t, err)
	var receivers []*nativeDeskFixture
	g02Check(t, json.Unmarshal(data, &receivers))
	if len(receivers) != 2 {
		t.Fatal("two selected receivers required")
	}
	for i, f := range receivers {
		name := []string{"~nec", "~bud"}[i]
		if f.ShipName != name || f.Desk != publisher.Desk || f.VereSHA != publisher.VereSHA || f.PID == publisher.PID || f.LensPort < 1024 || f.LensPort > 65535 || f.HTTPPort != 18082+i {
			t.Fatal("receiver identity changed")
		}
		f.client = publisher.client
		f.ledger = "receiver-" + strings.TrimPrefix(name, "~") + "-effects.jsonl"
		g02Check(t, f.check())
	}
	proof := map[string]any{"version": 2, "boundary": "native fake publisher and receivers; real receiver API/store/collector; fixture lifecycle backend"}
	save := func(name string, value any) {
		t.Helper()
		proof[name] = value
		b, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile("/lab/receiver-proof.json", b, 0600))
		t.Log(name)
	}
	before, err := publisher.Observe(ctx, publisher.Desk)
	g02Check(t, err)
	var expected deskwork.Observation
	data, err = os.ReadFile("/lab/receiver-source-expectation.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(data, &expected))
	if expected.Validate() != nil || before != expected {
		t.Fatal("publisher changed since its completed source workflow; no receiver effects permitted", before)
	}
	if !before.Exists || before.Publication != "published" {
		t.Fatal("publisher is not distributed", before)
	}
	source := deskwork.Publisher{Ship: "~zod", Desk: publisher.Desk, ExpectedHash: before.ClayHash}
	save("initial-publisher", before)
	wait := func(f *nativeDeskFixture, source deskwork.Publisher, label string) {
		t.Helper()
		for end := time.Now().Add(90 * time.Second); ; {
			ready, err := f.ReceiverSourceReady(ctx, f.Desk)
			g02Check(t, err)
			if ready {
				observed, err := f.Observe(ctx, f.Desk)
				g02Check(t, err)
				app, err := f.Application(ctx, f.Desk)
				g02Check(t, err)
				if deskwork.VerifyReceiver(source, observed, app) == nil {
					live := false
					for _, agent := range app.Agents {
						if agent.Name == "omabit-state" && agent.Live {
							live = true
						}
					}
					if live && app.FrontendState == "glob" && app.DocketHash == app.FrontendDocketHash {
						save(label+"-"+f.ship(), map[string]any{"source": source, "observation": observed, "application": app})
						return
					}
				}
			}
			if time.Now().After(end) {
				t.Fatal("receiver source/activation deadline", f.ship(), ready)
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
	content := []byte("<!doctype html><title>Omabit native fixture</title><p>Exact frozen frontend</p>\n")
	statuses := []func() protocol.DeskReceiverStatus{}
	for _, f := range receivers {
		observed, err := f.Observe(ctx, f.Desk)
		g02Check(t, err)
		sub, err := f.Subscription(ctx, f.Desk)
		g02Check(t, err)
		if observed.Exists || sub.Present {
			t.Fatal("receiver name already occupied", f.ship())
		}
		save("receiver-absence-"+f.ship(), map[string]any{"observation": observed, "subscription": sub})
		statuses = append(statuses, nativeReceiverInstall(t, ctx, f, source, save))
		wait(f, source, "initial")
		g02Check(t, nativeReceiverFrontend(ctx, f, content))
		save("frontend-initial-"+f.ship(), true)
	}
	// A new publisher commit is distributed through the existing subscriptions;
	// no second install and no receiver mutation are issued.
	g02Check(t, publisher.CommitFiles(ctx, publisher.Desk, before, []deskwork.DeltaFile{{Path: "hello.txt", Bytes: []byte("receiver rollout update\n")}}))
	after, err := publisher.Observe(ctx, publisher.Desk)
	g02Check(t, err)
	if after.Revision <= before.Revision || after.ClayHash == before.ClayHash || after.Publication != "published" {
		t.Fatal("publisher update not observed", after)
	}
	source.ExpectedHash = after.ClayHash
	save("updated-publisher", after)
	for i, f := range receivers {
		wait(f, source, "updated")
		status := statuses[i]()
		if !status.SourceReady || status.Observation.ClayHash != after.ClayHash {
			t.Fatal("updated receiver API source differs", status)
		}
		save("updated-status-"+f.ship(), status)
		g02Check(t, nativeReceiverFrontend(ctx, f, content))
		if f.effects.Load() != 1 {
			t.Fatal("receiver effect repeated")
		}
		save("frontend-updated-"+f.ship(), true)
	}
	save("ok", true)
}

func nativeReceiverInstall(t *testing.T, ctx context.Context, f *nativeDeskFixture, source deskwork.Publisher, save func(string, any)) func() protocol.DeskReceiverStatus {
	t.Helper()
	name := strings.TrimPrefix(f.ship(), "~")
	dir := "/lab/runner-" + name
	g02Check(t, os.Mkdir(dir, 0700))
	s, err := testOpenStore(dir)
	g02Check(t, err)
	t.Cleanup(func() { _ = s.close() })
	request := registration("receiver", "native-receiver", uint16(f.HTTPPort))
	request.FakeShip = f.ship()
	_, _, err = s.register("fixture", request)
	g02Check(t, err)
	instance, err := loadInstance(s.db, "receiver")
	g02Check(t, err)
	instance.Public.ObservedState = "running"
	instance.Public.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
	saveRecoveryFixture(t, s, instance)
	serverCtx, cancel := context.WithCancel(ctx)
	r := &Server{s: s, b: &nativeDeskBackend{&deskBackendFixture{}, f}, ctx: serverCtx, cancel: cancel, uid: uint32(os.Geteuid())}
	t.Cleanup(func() { _ = r.Close() })
	socket := "/lab/runner-" + name + ".sock"
	listener, err := net.Listen("unix", socket)
	g02Check(t, err)
	g02Check(t, os.Chmod(socket, 0600))
	server := &http.Server{Handler: r, ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
		return context.WithValue(ctx, peerKey{}, uint32(os.Geteuid()))
	}}
	go server.Serve(listener)
	t.Cleanup(func() { _ = server.Close() })
	registry, err := collection.Open("/lab/collector-" + name)
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "native", socket)
	g02Check(t, err)
	status := func() protocol.DeskReceiverStatus {
		t.Helper()
		out, err := registry.InspectDeskReceiver(ctx, "native", entry.RunnerID, "receiver", f.Desk)
		g02Check(t, err)
		return out
	}
	save("absent-status-"+f.ship(), status())
	plan, err := registry.PlanDeskReceiver(ctx, "native", entry.RunnerID, "receiver", f.Desk, "", source)
	g02Check(t, err)
	_, err = registry.InstallDeskReceiver(ctx, "native", entry.RunnerID, plan)
	g02Check(t, err)
	r.wg.Wait()
	blocked, err := registry.DeskOperation(ctx, "native", entry.RunnerID, plan.Intent.Intent.ID)
	g02Check(t, err)
	if blocked.Receipt.Steps[len(blocked.Receipt.Steps)-1].State != "blocked" || f.effects.Load() != 0 {
		t.Fatal("default gate dispatched", blocked)
	}
	save("default-block-"+f.ship(), blocked)
	policy, err := s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: plan.Intent.Intent.Target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	g02Check(t, err)
	plan, err = registry.PlanDeskReceiver(ctx, "native", entry.RunnerID, "receiver", f.Desk, "", source)
	g02Check(t, err)
	lost := name == "bud"
	f.loseReply.Store(lost)
	_, err = registry.InstallDeskReceiver(ctx, "native", entry.RunnerID, plan)
	g02Check(t, err)
	r.wg.Wait()
	receipt, err := registry.DeskOperation(ctx, "native", entry.RunnerID, plan.Intent.Intent.ID)
	g02Check(t, err)
	want := "observed"
	if lost {
		want = "uncertain"
	}
	if receipt.Receipt.Steps[len(receipt.Receipt.Steps)-1].State != want || f.effects.Load() != 1 {
		t.Fatal("install result differs", receipt, f.effects.Load())
	}
	save("initial-install-"+f.ship(), receipt)
	_, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "ship", Target: plan.Intent.Intent.Target.Ship(), Gates: apppolicy.Blocked()})
	g02Check(t, err)
	if lost {
		for end := time.Now().Add(90 * time.Second); ; {
			out := status()
			if out.SourceReady && deskwork.VerifyReceiver(source, *out.Observation, *out.Application) == nil {
				break
			}
			if time.Now().After(end) {
				t.Fatal("lost-reply source deadline")
			}
			time.Sleep(250 * time.Millisecond)
		}
		_, err = registry.ReconcileDeskOperation(ctx, "native", entry.RunnerID, plan.Intent.Intent.ID)
		g02Check(t, err)
		r.wg.Wait()
		receipt, err = registry.DeskOperation(ctx, "native", entry.RunnerID, plan.Intent.Intent.ID)
		g02Check(t, err)
		if receipt.Receipt.Steps[len(receipt.Receipt.Steps)-1].State != "observed" {
			t.Fatal("unobserved read-only recovery", receipt)
		}
		save("reconciled-after-revocation-"+f.ship(), receipt)
	}
	_, err = registry.InstallDeskReceiver(ctx, "native", entry.RunnerID, plan)
	g02Check(t, err)
	if f.effects.Load() != 1 {
		t.Fatal("exact replay repeated install")
	}
	save("received-status-"+f.ship(), status())
	return status
}
