//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func (s *DeskRuntime) Frontend(ctx context.Context, desk string) (deskwork.FrontendState, error) {
	var out deskwork.FrontendState
	request, err := deskwork.FrontendInventory(s.ship, desk)
	if err != nil {
		return out, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return out, err
	}
	body, _ := json.Marshal(reply)
	if deskwork.DecodeJSONOutput(body, &out) != nil || out.Validate() != nil {
		return deskwork.FrontendState{}, errors.New("frontend observation unavailable or unsupported")
	}
	return out, nil
}

func (s *DeskRuntime) FrontendHash(ctx context.Context, m deskwork.FrontendManifest, files []deskwork.SourceFile) (string, error) {
	request, err := deskwork.FrontendGlobHash(m, files)
	if err != nil {
		return "", err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(reply)
	var hash string
	if deskwork.DecodeJSONOutput(body, &hash) != nil || (deskwork.Observation{Exists: true, Revision: 1, ClayHash: hash, Publication: "private"}).Validate() != nil {
		return "", errors.New("frontend artifact hash unavailable or unsupported")
	}
	return hash, nil
}

// UploadFrontend uses one ephemeral owner session on the pinned private Eyre
// endpoint. It never retains a credential or retries login/upload/logout. Caller
// must hold the lifecycle/policy lock and journal dispatch before entry.
func (s *DeskRuntime) UploadFrontend(ctx context.Context, review deskwork.FrontendReview, manifest deskwork.FrontendManifest, files []deskwork.SourceFile) (transfer deskwork.FrontendTransfer, err error) {
	transfer.SessionCleanup = "not-created"
	unavailable := errors.New("frontend upload unavailable or uncertain; inspect its receipt before any further effect")
	if review.CheckConfiguredUpload() != nil || review.Ship != s.ship || manifest.Validate() != nil || manifest.Digest != review.ArtifactDigest {
		return transfer, unavailable
	}
	// Validates the complete bytes and terminal-NUL restriction before login.
	if _, e := deskwork.FrontendGlobHash(manifest, files); e != nil {
		return transfer, unavailable
	}
	code, e := s.call(ctx, deskwork.FrontendOwnerCode())
	if e != nil || !protocol.ValidLoginCode(strings.TrimSpace(code)) {
		return transfer, unavailable
	}
	owner, e := eyre.NewWithTransport(s.httpEndpoint, s.ship, deskHTTPTransport{runtime: s})
	if e != nil {
		return transfer, unavailable
	}
	defer owner.Close()
	transfer.SessionCleanup = "unconfirmed"
	defer func() {
		// A fresh cleanup context is bounded even after the upload deadline expired.
		clean, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if owner.Logout(clean) == nil {
			transfer.SessionCleanup = "confirmed"
		}
	}()
	raw := []byte(strings.TrimSpace(code))
	code = ""
	if owner.Login(ctx, raw) != nil {
		return transfer, unavailable
	}
	// Native Docket offers no atomic expected-revision parameter. Narrow the
	// interval after authentication and refuse drift; postflight verifies again.
	continuity, e := s.SourceIdentity(ctx)
	if e != nil || continuity != review.Continuity {
		return transfer, unavailable
	}
	before, e := s.Observe(ctx, review.Stage.Target.Desk)
	if e != nil || before != review.Observation {
		return transfer, unavailable
	}
	app, e := s.Application(ctx, review.Stage.Target.Desk)
	if e != nil {
		return transfer, unavailable
	}
	frontend, e := s.Frontend(ctx, review.Stage.Target.Desk)
	if e != nil || frontend != review.Frontend || !reflect.DeepEqual(app, review.Application) {
		return transfer, unavailable
	}
	after, e := s.Observe(ctx, review.Stage.Target.Desk)
	if e != nil || after != before {
		return transfer, unavailable
	}
	transfer.Attempted = true
	if owner.UploadDocket(ctx, review.Stage.Target.Desk, manifest, files) != nil {
		return transfer, unavailable
	}
	transfer.Acknowledged = true
	return transfer, nil
}
