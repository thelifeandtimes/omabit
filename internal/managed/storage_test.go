package managed

import (
	"encoding/json"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestConventionalPathsAreReservedIncludingAncestors(t *testing.T) {
	s := testStore(t)
	req := registration("one", "one", 18080)
	req.PierPath = "fakeships/project/zod/pier"
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"fakeships/project/zod/pier", "fakeships/project/zod/nested/pier", "fakeships/project/pier"} {
		req := registration("two", "two", 18081)
		req.PierPath = path
		if _, _, err := s.register("actor", req); err == nil {
			t.Fatal(path)
		}
	}
	req = registration("two", "two", 18081)
	req.PierPath = "fakeships/project/nec/pier"
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal("sibling piers refused", err)
	}
}

func TestDiscoveredEnrollmentAndMovedRelinkBindFilesystemIdentity(t *testing.T) {
	s := testStore(t)
	id := pierstore.Identity{Device: 1, Inode: 2, UrbDevice: 1, UrbInode: 3}
	source := pierstore.Candidate{Path: "fakeships/project/zod/pier", Kind: "fake", Fingerprint: pierstore.Fingerprint(id), Identity: id, MountIdentity: pierstore.DirectoryIdentity{Device: 1, Inode: 4, Birth: "1:0"}}
	req := protocol.EnrollPierRequest{ID: "adopted", Path: source.Path, Ship: "~zod", Fingerprint: source.Fingerprint, HTTPPort: 18080, RequestID: "enroll", ConfirmStopped: true}
	op, err := s.enrollPier("actor", req, source, "enroll")
	if err != nil || op.Status != "queued" {
		t.Fatal(op, err)
	}
	v, err := loadInstance(s.db, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Runtime.Adoption.Uncertain || !v.Public.IdentityPending || v.Runtime.Storage.Identity != id {
		t.Fatal("enrollment lost its quarantine or identity")
	}
	// Simulate completed stopped enrollment while preserving the real stored ID.
	v.Public.ActiveOperationID = ""
	v.Runtime.Adoption = nil
	v.Public.ObservedState = "missing"
	v.Public.RecoveryRequired = true
	data, _ := json.Marshal(v)
	if _, err := s.db.Exec("UPDATE instances SET body=? WHERE id=?", data, req.ID); err != nil {
		t.Fatal(err)
	}
	source.Path = "fakeships/another-project/zod/pier"
	req.Path = source.Path
	req.Relink = true
	req.ExpectedRevision = 1
	req.RequestID = "relink"
	op, err = s.enrollPier("actor", req, source, "relink")
	if err != nil || op.Kind != "relink-pier" || op.Status != "succeeded" {
		t.Fatal(op, err)
	}
	v, err = loadInstance(s.db, req.ID)
	if err != nil || v.Public.PierPath != source.Path || v.Public.RecoveryRequired {
		t.Fatal(v.Public, err)
	}
	source.Identity.Inode++
	req.RequestID = "replacement"
	source.Fingerprint = strings.Repeat("f", 64)
	if _, err := s.enrollPier("actor", req, source, "replacement"); err == nil {
		t.Fatal("replacement inode silently adopted")
	}
}
