package deskwork

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func frontendStateFixture() (AppState, FrontendState) {
	r := FrontendReference{Mode: "glob-ames", Base: "frontend-route", Ship: "~zod", GlobHash: "0v1", DocketHash: "0v2"}
	s := FrontendState{SourceRevision: 3, DocketAvailable: true, Stage: "glob", Declared: r, Current: r}
	a := AppState{Zest: "live", SourceRevision: 3, ExpectedAgents: []string{}, Agents: []AppAgent{}, DocketAvailable: true, DocketHash: "0v2", FrontendMode: "glob-ames", FrontendState: "glob", FrontendDocketHash: "0v2"}
	return a, s
}

func TestFrontendUploadRequiresCurrentOwnedGlob(t *testing.T) {
	a, s := frontendStateFixture()
	if err := CheckFrontendUpload("~zod", a, s); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFrontendMetadata("~zod", "frontend-route", "0v1", a, s); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"ship", "remote", "http", "stale", "revision", "absent", "suspend", "docket", "route", "hash"} {
		t.Run(change, func(t *testing.T) {
			a, s := frontendStateFixture()
			ship, base, hash := "~zod", "frontend-route", "0v1"
			switch change {
			case "ship":
				ship = "~nec"
			case "remote":
				a.SourceShip, a.SourceDesk = "~nec", "example"
			case "http":
				s.Declared.Mode, s.Current.Mode, a.FrontendMode = "glob-http", "glob-http", "glob-http"
				s.Declared.Ship, s.Current.Ship = "", ""
			case "stale":
				s.Current.GlobHash = "0v3"
			case "revision":
				a.SourceRevision++
			case "absent":
				s.Stage = "absent"
				s.Current = FrontendReference{Mode: "none"}
			case "suspend":
				a.FrontendState, s.Stage = "suspend", "suspend"
			case "docket":
				a.DocketHash = "0v3"
			case "route":
				base = "different"
			case "hash":
				hash = "0v3"
			}
			if VerifyFrontendMetadata(ship, base, hash, a, s) == nil {
				t.Fatal("changed frontend accepted")
			}
		})
	}
	a.FrontendState, s.Stage = "hung", "hung"
	if err := CheckFrontendUpload("~zod", a, s); err != nil {
		t.Fatal("initial failed glob must be replaceable", err)
	}
	if VerifyFrontendMetadata("~zod", "frontend-route", "0v1", a, s) == nil {
		t.Fatal("failed initial glob reported verified")
	}
}

func TestFrontendHashCarriesTypedBytesAsData(t *testing.T) {
	m, files := frontendFixture(t)
	q, err := FrontendGlobHash(m, files)
	if err != nil {
		t.Fatal(err)
	}
	body, err := q.Body()
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Source struct {
			Hoon struct {
				Code string `json:"code"`
				Next struct {
					Data string `json:"data"`
				} `json:"next"`
			} `json:"hoon"`
		} `json:"source"`
		Sink map[string]any `json:"sink"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	var rows []json.RawMessage
	if err = json.Unmarshal([]byte(envelope.Source.Hoon.Next.Data), &rows); err != nil || len(rows) != len(files) {
		t.Fatal("missing input rows", err)
	}
	for i, row := range rows {
		var columns []json.RawMessage
		if json.Unmarshal(row, &columns) != nil || len(columns) != 3 {
			t.Fatal("untyped glob row")
		}
		var encoded string
		var contentType string
		if json.Unmarshal(columns[1], &contentType) != nil || contentType != m.Files[i].ContentType {
			t.Fatal("native multipart MIME path atom changed")
		}
		if json.Unmarshal(columns[2], &encoded) != nil || encoded != base64.StdEncoding.EncodeToString(files[i].Bytes) || strings.Contains(envelope.Source.Hoon.Code, files[i].Path) {
			t.Fatal("asset lost or interpolated into source")
		}
	}
	if _, ok := envelope.Sink["stdout"]; !ok || len(envelope.Sink) != 1 {
		t.Fatal("artifact hashing must be read only")
	}
}
