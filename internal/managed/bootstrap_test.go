package managed

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestBootstrapSchemaUpgradePreservesLegacySelection(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("legacy", "legacy", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	// Schema-12 fake/comet records omitted selection and used the pinned preset.
	v.Runtime.Bootstrap = nil
	v.Public.Bootstrap = nil
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = saveInstance(tx, v); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if _, err = tx.Exec("PRAGMA user_version=12"); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = s.initialize(); err != nil {
		t.Fatal(err)
	}
	got, err := loadInstance(s.db, "legacy")
	if err != nil || !reflect.DeepEqual(got, v) {
		t.Fatal("legacy bootstrap/runtime record changed", err)
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 13 {
		t.Fatal(version, err)
	}
}

func TestNewFakeAndCometBootstrapIsStagedImmutableAndReplaySafe(t *testing.T) {
	for _, comet := range []bool{false, true} {
		t.Run(fmt.Sprint(comet), func(t *testing.T) {
			s := testStore(t)
			s.mode = DevelopmentMode
			bare := registration("bare", "bare", 18080)
			if _, _, err := s.registerKind("actor", bare, nil, comet); err != nil {
				t.Fatal(err)
			}
			v, err := loadInstance(s.db, "bare")
			if err != nil || v.Runtime.Bootstrap == nil || v.Runtime.Bootstrap.Kind != "bare" || v.Runtime.Boot != nil || v.Public.Bootstrap.Kind != "bare" {
				t.Fatal("default boot is not canonical", err)
			}
			body := "selected test pill"
			p := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(body))), Bytes: int64(len(body))}
			req := registration("recommended", "recommended", 18081)
			req.Bootstrap = &protocol.Bootstrap{Kind: "recommended", Pill: &p}
			_, _, err = s.registerKind("actor", req, nil, comet)
			requireFault(t, err, "pill_unavailable")
			r := &Server{s: s, ctx: context.Background()}
			upload := httptest.NewRequest("POST", protocol.PillUploadPath, strings.NewReader(body))
			upload.Header.Set("Content-Type", "application/octet-stream")
			upload.Header.Set(protocol.PillHashHeader, p.SHA256)
			w := httptest.NewRecorder()
			r.stagePill(w, upload, "operator")
			if w.Code != 201 {
				t.Fatal(w.Body)
			}
			op, hit, err := s.registerKind("actor", req, nil, comet)
			if err != nil || hit {
				t.Fatal(err)
			}
			v, err = loadInstance(s.db, req.ID)
			if err != nil || v.Runtime.Bootstrap.Pill == nil || *v.Runtime.Bootstrap.Pill != p || v.Public.Bootstrap.Kind != "recommended" || v.Runtime.Boot != nil {
				t.Fatal("selected bootstrap lost or published Ames", err)
			}
			if err = os.Remove(bootpill.CachePath(s.directory, p)); err != nil {
				t.Fatal(err)
			}
			again, hit, err := s.registerKind("actor", req, nil, comet)
			if err != nil || !hit || again.ID != op.ID {
				t.Fatal("replay required uploading the pill again", err)
			}
			req.Bootstrap = &protocol.Bootstrap{Kind: "bare"}
			_, _, err = s.registerKind("actor", req, nil, comet)
			requireFault(t, err, "request_conflict")
			for _, b := range []*protocol.Bootstrap{{Kind: "recommended"}, {Kind: "bare", Pill: &p}, {Kind: "unknown"}} {
				req.RequestID = "bad"
				req.Bootstrap = b
				_, _, err = s.registerKind("actor", req, nil, comet)
				requireFault(t, err, "invalid_request")
			}
		})
	}
}
