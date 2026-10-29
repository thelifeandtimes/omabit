package managed

import (
	"errors"
	"os"
	"testing"
)

func TestValidationSnapshotChecksChangedRowsInSameTransaction(t *testing.T) {
	fixture := originalPrivate45(t)
	s := g02Open(t, g02Materialize(t, fixture))
	defer s.close()
	tx, err := s.db.Begin()
	g02Check(t, err)
	defer tx.Rollback()
	g02Check(t, validatePrivateWorkspaces(tx))
	_, err = tx.Exec("UPDATE private_project_workspaces SET body='{}'")
	g02Check(t, err)
	if validatePrivateWorkspaces(tx) == nil {
		t.Fatal("prior validation hid changed journal in same transaction")
	}
	g02Check(t, tx.Rollback())
	g02Check(t, validatePrivateWorkspaces(s.db))
}

func TestValidationSnapshotSharesOnlyOneTraversal(t *testing.T) {
	dir := t.TempDir()
	g02Check(t, os.Chmod(dir, 0700))
	s := g02Open(t, dir)
	defer s.close()
	tx, err := s.db.Begin()
	g02Check(t, err)
	defer tx.Rollback()
	calls := 0
	child := func(q querier) error { calls++; return nil }
	parent := func(q querier) error {
		q = projectRecoveryReadOnly(q)
		for n := 0; n < 5; n++ {
			if err := validateSnapshot(q, "child", child); err != nil {
				return err
			}
		}
		return nil
	}
	g02Check(t, validateSnapshot(tx, "parent", parent))
	if calls != 1 {
		t.Fatalf("repeated predecessor validation: %d", calls)
	}
	g02Check(t, validateSnapshot(tx, "parent", parent))
	if calls != 2 {
		t.Fatal("memo escaped the traversal")
	}
	// An ordinary database gets a new read transaction per traversal.
	g02Check(t, tx.Rollback())
	g02Check(t, validateSnapshot(s.db, "parent", parent))
	if calls != 3 {
		t.Fatal("database validation did not use a read snapshot")
	}
	tx, err = s.db.Begin()
	g02Check(t, err)
	defer tx.Rollback()
	sentinel := errors.New("invalid dependency")
	leaf := func(q querier) error { return sentinel }
	if !errors.Is(validateSnapshot(tx, "parent", func(q querier) error { return validateSnapshot(q, "child", leaf) }), sentinel) {
		t.Fatal("validation failure suppressed")
	}
	var cycle func(querier) error
	cycle = func(q querier) error { return validateSnapshot(q, "cycle", cycle) }
	if validateSnapshot(tx, "cycle", cycle) == nil {
		t.Fatal("cycle accepted")
	}
}
