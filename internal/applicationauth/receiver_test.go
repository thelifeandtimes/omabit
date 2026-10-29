package applicationauth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func artifactFixture(t *testing.T, script string) (*Artifact, string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"id":"fixture.auth-lab"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "application-profile.json"), []byte(`{"version":1,"profile":"auth-lab"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "receiver.py"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"HOME": t.TempDir(), "XDG_STATE_HOME": t.TempDir()}
	a, err := OpenReceiver(root, "receiver.py", []string{"manifest.json", "application-profile.json", "receiver.py"}, env)
	if err != nil {
		t.Fatal(err)
	}
	return a, root, env
}

func TestRetainedReceiverSurvivesSourceUpdateAndUsesPrivateInput(t *testing.T) {
	script := `import json,os,sys
value=json.load(sys.stdin)
assert value['secret']=='fixture-private-cookie'
assert 'PYTHONPATH' not in os.environ and 'LD_PRELOAD' not in os.environ
assert all('fixture-private-cookie' not in part for part in sys.argv)
print(json.dumps({'ok':True,'reconcile':sys.argv[1:]==['--reconcile']}))
`
	a, root, env := artifactFixture(t, script)
	t.Setenv("PYTHONPATH", "/must/not/inherit")
	cache := filepath.Join(t.TempDir(), "generations")
	retained, err := a.Retain(context.Background(), cache)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Target() != a.Target() {
		t.Fatal("retention changed approved identity")
	}
	if _, err := a.Retain(context.Background(), cache); err != nil {
		t.Fatal("idempotent retention", err)
	}
	if err := os.WriteFile(filepath.Join(root, "receiver.py"), []byte("raise Exception('unapproved')"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.Verify(context.Background()); !errors.Is(err, ErrArtifact) {
		t.Fatal("mutable source accepted", err)
	}
	loaded, err := LoadReceiver(cache, a.Target().SHA256, env)
	if err != nil {
		t.Fatal(err)
	}
	for _, recovery := range []bool{false, true} {
		var out []byte
		if recovery {
			out, err = loaded.Reconcile(context.Background(), []byte(`{"secret":"fixture-private-cookie"}`))
		} else {
			out, err = loaded.Deliver(context.Background(), []byte(`{"secret":"fixture-private-cookie"}`))
		}
		var value map[string]bool
		if err != nil || json.Unmarshal(out, &value) != nil || !value["ok"] || value["reconcile"] != recovery {
			t.Fatal("private receiver failed", err)
		}
	}
	// The retained copy is still verified before every launch.
	path := filepath.Join(cache, a.Target().SHA256, "receiver.py")
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("print('changed')"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loaded.Deliver(context.Background(), nil); !errors.Is(err, ErrArtifact) {
		t.Fatal("tampered retained receiver ran", err)
	}
}

func TestReceiverBoundsFailureOutputAndLifetime(t *testing.T) {
	for name, script := range map[string]string{
		"oversized": "print('private' * 10000)",
		"exception": "raise Exception('private token must not escape')",
		"timeout":   "import time; time.sleep(60)",
	} {
		t.Run(name, func(t *testing.T) {
			a, _, _ := artifactFixture(t, script)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			start := time.Now()
			out, err := a.Deliver(ctx, []byte("private"))
			if !errors.Is(err, ErrDelivery) || len(out) != 0 || strings.Contains(err.Error(), "token") || time.Since(start) > 3*time.Second {
				t.Fatal("unbounded/private failure", err, time.Since(start))
			}
		})
	}
}

func TestReceiverRefusesLinksTraversalAndUnapprovedEnvironment(t *testing.T) {
	a, root, env := artifactFixture(t, "print('{}')")
	for _, files := range [][]string{{"../receiver.py"}, {"receiver.py", "manifest.json", "receiver.py"}, {"manifest.json"}} {
		if _, err := OpenReceiver(root, "receiver.py", files, env); !errors.Is(err, ErrArtifact) {
			t.Fatal(err)
		}
	}
	env["PYTHONPATH"] = "/other"
	if _, err := OpenReceiver(root, "receiver.py", a.names, env); !errors.Is(err, ErrArtifact) {
		t.Fatal("accepted code-injection environment", err)
	}
	delete(env, "PYTHONPATH")
	path := filepath.Join(root, "receiver.py")
	if err := os.Rename(path, path+".real"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path+".real", path); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReceiver(root, "receiver.py", a.names, env); !errors.Is(err, ErrArtifact) {
		t.Fatal("accepted symlink", err)
	}
}

type reconcileFixture struct {
	*receiver
	calls     int
	loseReply bool
}

func (r *reconcileFixture) Reconcile(_ context.Context, data []byte) ([]byte, error) {
	r.calls++
	if r.loseReply {
		return nil, ErrDelivery
	}
	var request map[string]any
	_ = json.Unmarshal(data, &request)
	delete(request, "action")
	delete(request, "origin")
	delete(request, "ship")
	request["status"], request["outcome"] = "reconciled", "removed"
	return json.Marshal(request)
}

func TestReconciliationRequiresClosedSessionAndDurableMatchingReceipt(t *testing.T) {
	ctx := context.Background()
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	if _, err := f.e.Connect(ctx, a.ID, r); err != nil {
		t.Fatal(err)
	}
	receiver := &reconcileFixture{receiver: r}
	if _, err := f.e.Reconcile(ctx, a.ID, receiver); !errors.Is(err, ErrRecovery) || receiver.calls != 0 {
		t.Fatal("reconciled active session", err)
	}
	if _, err := f.e.Disconnect(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	receiver.loseReply = true
	if got, err := f.e.Reconcile(ctx, a.ID, receiver); !errors.Is(err, ErrRecovery) || !got.ReceiverReconciliation {
		t.Fatal(got, err)
	}
	receiver.loseReply = false
	f.reopen(t)
	got, err := f.e.Reconcile(ctx, a.ID, receiver)
	if err != nil || got.ReceiverReconciliation || got.ReceiverOutcome != "removed" {
		t.Fatal(got, err)
	}
	if _, err := f.e.Reconcile(ctx, a.ID, receiver); err != nil || receiver.calls != 2 {
		t.Fatal("repeated receiver cleanup after durable ack", err)
	}
	_ = f.approve(t, r) // exact old cleanup fence was cleared, so a new approval is possible
}

func TestReceiverReapsItsOwnedProcessGroupAfterParentExit(t *testing.T) {
	a, _, env := artifactFixture(t, `import subprocess,os
p=subprocess.Popen(['/usr/bin/python3','-c','import time;time.sleep(60)'])
with open(os.path.join(os.environ['HOME'],'child.pid'),'w') as f: f.write(str(p.pid))
print('{}')
`)
	start := time.Now()
	if _, err := a.Deliver(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(env["HOME"], "child.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid := strings.TrimSpace(string(data))
	if pid == "" || strings.Trim(pid, "0123456789") != "" {
		t.Fatal("invalid fixture child")
	}
	for time.Since(start) < 2*time.Second {
		state, err := os.ReadFile("/proc/" + pid + "/stat")
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		index := strings.LastIndex(string(state), ") ")
		if err == nil && index >= 0 && index+2 < len(state) && (state[index+2] == 'Z' || state[index+2] == 'X') {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fixture child survived receiver completion")
}
