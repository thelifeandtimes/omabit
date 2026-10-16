package managed

import (
	"context"
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
	if v.Runtime.Boot != nil || v.Runtime.AmesPort != 0 || v.Public.AmesPublished {
		t.Fatal("fake enrollment enabled host Ames publication")
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

type catalogueBackend struct {
	fakeBackend
	sources []pierstore.Candidate
}

func (b *catalogueBackend) PierSources(context.Context) ([]pierstore.Candidate, error) {
	return append([]pierstore.Candidate(nil), b.sources...), nil
}

// After a reboot renumbers st_dev, discovery must keep recognising the
// registered pier instead of reporting a replaced directory.
func TestCatalogueToleratesDeviceRenumberingAfterReboot(t *testing.T) {
	s := testStore(t)
	stored := pierstore.Identity{Device: 30, Inode: 2, UrbDevice: 30, UrbInode: 3, Birth: "10:1", UrbBirth: "10:2"}
	mount := pierstore.DirectoryIdentity{Device: 30, Inode: 4, Birth: "10:0"}
	source := pierstore.Candidate{Path: "fakeships/project/zod/pier", Kind: "fake", Fingerprint: pierstore.Fingerprint(stored), Identity: stored, MountIdentity: mount}
	req := protocol.EnrollPierRequest{ID: "rebooted", Path: source.Path, Ship: "~zod", Fingerprint: source.Fingerprint, HTTPPort: 18080, RequestID: "enroll", ConfirmStopped: true}
	if _, err := s.enrollPier("actor", req, source, "enroll"); err != nil {
		t.Fatal(err)
	}
	after := source
	after.Identity.Device, after.Identity.UrbDevice, after.MountIdentity.Device = 31, 31, 31
	after.Fingerprint = pierstore.Fingerprint(after.Identity)
	r := &Server{s: s, b: &catalogueBackend{sources: []pierstore.Candidate{after}}}
	c := r.pierCatalogue(context.Background())
	if c == nil || c.Error != "" || len(c.Candidates) != 1 {
		t.Fatal(c)
	}
	if got := c.Candidates[0]; got.Problem != "" || got.EnrolledBy != "rebooted" || got.Moved {
		t.Fatalf("renumbered device misreported: %+v", got)
	}
	replaced := after
	replaced.Identity.Birth = "11:0"
	r.b = &catalogueBackend{sources: []pierstore.Candidate{replaced}}
	if got := r.pierCatalogue(context.Background()).Candidates[0]; got.Problem == "" {
		t.Fatal("replaced directory was not flagged")
	}
}
