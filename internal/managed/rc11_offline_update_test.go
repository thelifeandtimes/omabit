package managed

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func rc11OfflineDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"runner.sqlite", ".lock"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "runner.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/offline-schema19.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return dir, db
}

func TestRC11OfflineLockChild(t *testing.T) {
	dir := os.Getenv("OMABIT_OFFLINE_LOCK_CHILD")
	if dir == "" {
		return
	}
	lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	fmt.Fprintln(os.Stdout, "locked")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestRC11OfflineConflictingProcessAndInheritedLock(t *testing.T) {
	dir, _ := rc11OfflineDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRC11OfflineLockChild$")
	cmd.Env = append(os.Environ(), "OMABIT_OFFLINE_LOCK_CHILD="+dir)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { in.Close(); _ = cmd.Wait() }()
	ready, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || ready != "locked\n" {
		t.Fatal(ready, err)
	}
	if lock, err := LockOfflineState(dir, uint32(os.Geteuid())); err == nil {
		lock.Close()
		t.Fatal("different process ownership was bypassed")
	}
	if err := in.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	parent, err := LockOfflineState(dir, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	fd, err := syscall.Dup(int(parent.File().Fd()))
	if err != nil {
		t.Fatal(err)
	}
	child, err := AdoptOfflineStateLock(dir, uint32(os.Geteuid()), os.NewFile(uintptr(fd), "inherited-lock"))
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if other, err := LockOfflineState(dir, uint32(os.Geteuid())); err == nil {
		other.Close()
		t.Fatal("helper close unlocked its parent")
	}
}

func TestRC11OfflineExistingStateLockAndReadOnlyInspection(t *testing.T) {
	dir, db := rc11OfflineDB(t)
	lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if other, err := LockOfflineState(dir, uint32(os.Geteuid())); err == nil {
		other.Close()
		t.Fatal("runtime flock was not exclusive")
	}
	state, err := lock.Inspect(context.Background())
	if err != nil || state.Schema != 19 || state.RunnerID != "runner-"+strings.Repeat("a", 32) || state.Token != "" {
		t.Fatal(state, err)
	}
	var schema int
	if err := db.QueryRow("PRAGMA user_version").Scan(&schema); err != nil || schema != 19 {
		t.Fatal(schema, err)
	}
	if _, err := db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Inspect(context.Background()); err == nil {
		t.Fatal("future schema was accepted")
	}
}

func TestRC11OfflineRejectsActiveAndUncertainStateWithoutReconciliation(t *testing.T) {
	for _, mutation := range []string{
		`UPDATE operations SET body=json_set(body,'$.status','queued')`,
		`UPDATE operations SET body=json_set(body,'$.status','running')`,
		`UPDATE instances SET body=json_set(body,'$.public.active_operation_id','op-unknown') WHERE id='stopped'`,
		`UPDATE instances SET body=json_set(body,'$.runtime.maintenance_uncertain',json('true')) WHERE id='stopped'`,
		`UPDATE instances SET body=json_set(body,'$.runtime.runtime.containers[0].create_uncertain',json('true')) WHERE id='running'`,
		`UPDATE instances SET body=json_set(body,'$.public.recovery_required',json('true')) WHERE id='stopped'`,
	} {
		t.Run(mutation, func(t *testing.T) {
			dir, db := rc11OfflineDB(t)
			if _, err := db.Exec(mutation); err != nil {
				t.Fatal(err)
			}
			lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			if _, err := lock.Inspect(context.Background()); err == nil {
				t.Fatal("unresolved state accepted")
			}
			var schema int
			db.QueryRow("PRAGMA user_version").Scan(&schema)
			if schema != 19 {
				t.Fatal("inspection migrated state")
			}
		})
	}
}

func TestRC11OfflineMigrationAndFenceAreOnePreservingTransaction(t *testing.T) {
	if StateSchemaVersion < 20 {
		t.Fatal("offline upgrade requires the integrated schema20 fence")
	}
	dir, db := rc11OfflineDB(t)
	lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	before := map[string]string{}
	for _, table := range []string{"instances", "operations", "maintenance_policies"} {
		var text string
		if err := db.QueryRow("SELECT group_concat(hex(body),'|') FROM " + table).Scan(&text); err != nil {
			t.Fatal(err)
		}
		before[table] = text
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_fence BEFORE INSERT ON meta WHEN new.key='management_update_v1' BEGIN SELECT RAISE(ABORT,'injected fence failure'); END`); err != nil {
		t.Fatal(err)
	}
	runner, token := "runner-"+strings.Repeat("a", 32), strings.Repeat("e", 64)
	if _, err := lock.Fence(context.Background(), runner, token); err == nil {
		t.Fatal("injected failure was ignored")
	}
	var schema, count int
	db.QueryRow("PRAGMA user_version").Scan(&schema)
	db.QueryRow("SELECT count(*) FROM meta WHERE key=?", managementFenceKey).Scan(&count)
	if schema != 19 || count != 0 {
		t.Fatal("migration and fence were not atomic", schema, count)
	}
	if _, err := db.Exec("DROP TRIGGER fail_fence"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		state, err := lock.Fence(context.Background(), runner, token)
		if err != nil || !state.Quiesced || state.Token != token || state.Schema != StateSchemaVersion {
			t.Fatal(state, err)
		}
	}
	if _, err := lock.Fence(context.Background(), runner, strings.Repeat("f", 64)); err == nil {
		t.Fatal("foreign journal token accepted")
	}
	for table, want := range before {
		var got string
		if err := db.QueryRow("SELECT group_concat(hex(body),'|') FROM " + table).Scan(&got); err != nil || got != want {
			t.Fatal("rewrote historical bytes", table, err)
		}
	}
	if err := lock.Verify(); err != nil {
		t.Fatal("changed state ownership", err)
	}
}

func TestRC11OfflinePreservesProvenPermanentMovedFence(t *testing.T) {
	dir, db := rc11OfflineDB(t)
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	h := protocol.Handoff{ID: strings.Repeat("a", 32), Direction: "out", State: "moved", SourceRunner: "runner-" + strings.Repeat("a", 32), SourceInstance: "stopped", DestinationRunner: "runner-" + strings.Repeat("b", 32), DestinationInstance: "arrived", Ship: "~zod", Mode: "fake", VerifyKey: hex.EncodeToString(key.Public().(ed25519.PublicKey))}
	h.Commitment = hex.EncodeToString(ed25519.Sign(key, h.CommitmentBytes()))
	data, _ := json.Marshal(h)
	if _, err := db.Exec("UPDATE instances SET body=json_set(body,'$.runtime.handoff',json(?)) WHERE id='stopped'", string(data)); err != nil {
		t.Fatal(err)
	}
	var before string
	db.QueryRow("SELECT body FROM instances WHERE id='stopped'").Scan(&before)
	lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err := lock.Inspect(context.Background()); err != nil {
		t.Fatal("completed permanent fence blocked upgrade", err)
	}
	var after string
	db.QueryRow("SELECT body FROM instances WHERE id='stopped'").Scan(&after)
	if after != before {
		t.Fatal("inspection cleared permanent source fence")
	}
	if _, err := db.Exec("UPDATE instances SET body=json_set(body,'$.runtime.handoff.commitment','invalid') WHERE id='stopped'"); err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Inspect(context.Background()); err == nil {
		t.Fatal("unproven transfer accepted")
	}
}
