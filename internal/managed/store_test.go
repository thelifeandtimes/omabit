package managed

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func privateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func testStore(t *testing.T) *store {
	t.Helper()
	s, err := openStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.close() })
	return s
}

func registration(id, key string, port uint16) protocol.CreateFakeRequest {
	return protocol.CreateFakeRequest{ID: id, RequestID: key, Config: protocol.Configuration{HTTPPort: port, AmesPort: 34343, Loom: 31}}
}

func requireFault(t *testing.T, err error, code string) {
	t.Helper()
	var f *fault
	if !errors.As(err, &f) || f.problem.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

func TestDurableIdentityRevisionAndReplay(t *testing.T) {
	dir := privateDir(t)
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := s.runnerID
	op, replayed, err := s.register("local-uid:1000", registration("alpha", "register", 18080))
	if err != nil || replayed {
		t.Fatalf("register: %+v %v", op, err)
	}
	_, _, err = s.register("local-uid:1000", registration("bravo", "register", 18081))
	requireFault(t, err, "request_conflict")
	_, _, err = s.register("local-uid:2000", registration("bravo", "register", 18081))
	if err != nil {
		t.Fatal(err)
	}
	config := protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18082, AmesPort: 34343, Loom: 31}}
	if _, err := s.configure("local-uid:1000", "alpha", config); err != nil {
		t.Fatal(err)
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	if s.runnerID != id {
		t.Fatal("runner identity changed")
	}
	retry, found, err := s.register("local-uid:1000", registration("alpha", "register", 18080))
	if err != nil || !found || retry.ID != op.ID {
		t.Fatalf("durable replay: %+v %v", retry, err)
	}
	_, err = s.configure("local-uid:1000", "alpha", config)
	requireFault(t, err, "revision_conflict")
	snapshot, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Instances) != 2 || snapshot.Instances[0].Config.HTTPPort != 18082 || snapshot.Instances[0].ConfigRevision != 2 {
		t.Fatalf("lost state: %+v", snapshot)
	}
	for _, name := range []string{"runner.sqlite", "runner.sqlite-wal", "runner.sqlite-shm"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("database permissions: %s %v", name, info.Mode())
		}
	}
}

func TestConcurrentRegistrationAndConfiguration(t *testing.T) {
	s := testStore(t)
	const n = 12
	var wg sync.WaitGroup
	ops := make(chan protocol.Operation, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, _, err := s.register("actor", registration("alpha", "same", 18080))
			if err != nil {
				t.Error(err)
				return
			}
			ops <- op
		}()
	}
	wg.Wait()
	close(ops)
	id := ""
	for op := range ops {
		if id != "" && id != op.ID {
			t.Fatal("duplicate registration")
		}
		id = op.ID
	}
	winners := make(chan bool, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}})
			if err != nil {
				var f *fault
				if !errors.As(err, &f) || f.problem.Code != "revision_conflict" {
					t.Error(err)
				}
			}
			winners <- err == nil
		}()
	}
	wg.Wait()
	close(winners)
	wins := 0
	for win := range winners {
		if win {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("revision winners=%d", wins)
	}
}

func TestJournalInterruptionDoesNotReexecute(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	req := protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1}
	op, _, err := s.accept("actor", "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.beginOperation(op.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	old, replay, err := s.accept("actor", "alpha", req)
	if err != nil || !replay || old.ID != op.ID || old.Status != "interrupted" {
		t.Fatalf("interrupted replay: %+v %v", old, err)
	}
	_, _, err = s.accept("actor", "alpha", protocol.OperationRequest{RequestID: "different", Kind: "start", ExpectedRevision: 1})
	requireFault(t, err, "invalid_state")
	_, _, err = s.accept("actor", "alpha", protocol.OperationRequest{RequestID: "stop", Kind: "stop", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPortReservationsIncludePendingAppliedRuntime(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	// An old port remains reserved while a container using it still exists,
	// even if the desired configuration has moved to another port.
	v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{Name: "old", Removed: false})
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := saveInstance(tx, v); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}}); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.register("actor", registration("bravo", "reg-b", 18080))
	requireFault(t, err, "port_conflict")
}

func TestUnsafeStoreAndCorruptionRefused(t *testing.T) {
	dir := privateDir(t)
	if err := os.WriteFile(filepath.Join(dir, "runner.sqlite"), []byte("not SQLite"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := openStore(dir); err == nil {
		s.close()
		t.Fatal("corrupt store accepted")
	}
	data, err := os.ReadFile(filepath.Join(dir, "runner.sqlite"))
	if err != nil || string(data) != "not SQLite" {
		t.Fatal("corrupt input replaced")
	}
	other := privateDir(t)
	if err := os.Symlink("elsewhere", filepath.Join(other, "runner.sqlite-wal")); err != nil {
		t.Fatal(err)
	}
	if s, err := openStore(other); err == nil {
		s.close()
		t.Fatal("symlink sidecar accepted")
	}
}

func TestStoreSurvivesProcessDeath(t *testing.T) {
	dir := privateDir(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStoreCrashHelper$")
	cmd.Env = []string{"URBIT_STORE_CRASH_TEST=" + dir}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	line, err := bufio.NewReader(pipe).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "READY" {
		t.Fatalf("helper: %q %v", line, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	v, err := loadInstance(s.db, "alpha")
	if err != nil || v.Public.ConfigRevision != 1 {
		t.Fatalf("uncommitted edit survived: %+v %v", v, err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	op, replay, err := s.accept("actor", "alpha", protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1})
	if err != nil || !replay || op.Status != "interrupted" {
		t.Fatalf("committed operation lost: %+v %v", op, err)
	}
}

func TestStoreCrashHelper(t *testing.T) {
	dir := os.Getenv("URBIT_STORE_CRASH_TEST")
	if dir == "" {
		return
	}
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.register("actor", registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.accept("actor", "alpha", protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(tx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	v.Public.ConfigRevision = 99
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("UPDATE instances SET body=? WHERE id='alpha'", data); err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.WriteString("READY\n")
	for {
		time.Sleep(time.Hour)
	}
}
