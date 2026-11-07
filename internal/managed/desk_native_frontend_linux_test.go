//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func (f *nativeDeskFixture) recordEffect(id int32, state string) error {
	name := "workflow-effects.jsonl"
	if f.ledger != "" {
		name = f.ledger
	}
	file, err := os.OpenFile("/lab/"+name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err = json.NewEncoder(file).Encode(map[string]any{"id": id, "state": state}); err != nil {
		return err
	}
	return file.Sync()
}

// Native Docket's existing upload workflow, confined to the admitted fake ship.
// No caller URL, credential, filesystem path or production session is accepted.
// This is qualification, not an agent-facing upload adapter.
func (f *nativeDeskFixture) uploadGlob(ctx context.Context, content []byte) error {
	if err := f.check(); err != nil {
		return err
	}
	files := []deskwork.SourceFile{
		{Path: "assets/Upper_case.file.png", Bytes: []byte{0, 1, 0, 255, 42}},
		{Path: "empty.txt", Bytes: []byte{}},
		{Path: "index.html", Bytes: content},
	}
	entries := make([]deskwork.File, 0, len(files))
	for _, file := range files {
		h := sha256.Sum256(file.Bytes)
		entries = append(entries, deskwork.File{Path: file.Path, Bytes: int64(len(file.Bytes)), SHA256: hex.EncodeToString(h[:])})
	}
	source, err := deskwork.NewManifest(entries)
	if err != nil {
		return err
	}
	artifact, err := deskwork.NewFrontendManifest(source)
	if err != nil {
		return err
	}
	q, err := deskwork.FrontendGlobHash(artifact, files)
	var expectedHash string
	if err = f.decode(ctx, q, err, &expectedHash); err != nil {
		return err
	}
	// Independently observed from native45, with exact bytes/types confirmed by
	// read-only native46/47 inspection. Catch a changed noun representation before
	// another upload, not only by inspecting the resulting source commit.
	if expectedHash != "0v2.bat46.e07k7.hgk43.vkgeg.oc0i7" {
		return errors.New("fixture artifact hash differs from recorded native glob")
	}
	readState := func() (deskwork.FrontendState, error) {
		var state deskwork.FrontendState
		q, err := deskwork.FrontendInventory(f.ship(), f.Desk)
		if err = f.decode(ctx, q, err, &state); err != nil {
			return state, err
		}
		return state, state.Validate()
	}
	before, err := readState()
	if err != nil {
		return err
	}
	app, err := f.Application(ctx, f.Desk)
	if err != nil {
		return err
	}
	if err = deskwork.CheckFrontendUpload(f.ship(), app, before); err != nil {
		return err
	}
	owner, err := eyre.New("http://127.0.0.1:18081", "~zod")
	if err != nil {
		return err
	}
	defer owner.Close()
	if err = owner.Login(ctx, []byte("lidlut-tabwed-pillex-ridrup")); err != nil {
		return err
	}
	defer owner.Logout(context.Background())
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
		return errors.New("fixture session unavailable")
	}
	id := f.effects.Add(1)
	if err = f.recordEffect(id, "dispatched"); err != nil {
		return err
	}
	if err = owner.UploadDocket(ctx, f.Desk, artifact, files); err != nil {
		return err
	}
	if err = f.check(); err != nil {
		return err
	}
	if err = f.recordEffect(id, "acknowledged"); err != nil {
		return err
	}
	var after deskwork.FrontendState
	for end := time.Now().Add(20 * time.Second); ; {
		after, err = readState()
		if err != nil {
			return err
		}
		app, err = f.Application(ctx, f.Desk)
		if err != nil {
			return err
		}
		if deskwork.VerifyFrontendMetadata(f.ship(), before.Declared.Base, expectedHash, app, after) == nil && after.SourceRevision > before.SourceRevision {
			break
		}
		if time.Now().After(end) {
			proof, _ := json.MarshalIndent(map[string]any{"expected_hash": expectedHash, "before": before, "after": after, "app": app}, "", "  ")
			_ = os.WriteFile("/lab/frontend-failed-proof.json", proof, 0600)
			return errors.New("frontend hash or source commit not observed")
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Verify every asset and its MIME type, including an empty file and embedded
	// zero bytes. HTTP 200 alone accepts Docket's missing-file index fallback.
	for i, file := range files {
		req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:18081/apps/"+after.Declared.Base+"/"+file.Path, nil)
		if err != nil {
			return err
		}
		req.AddCookie(&http.Cookie{Name: session.Name, Value: session.Value})
		resp, err := f.client.Do(req)
		if err != nil {
			return errors.New("fixture frontend read unavailable")
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != artifact.Files[i].ContentType || !bytes.Equal(data, file.Bytes) {
			return errors.New("fixture served frontend differs from frozen content: " + file.Path)
		}
	}
	proof, err := json.MarshalIndent(map[string]any{"ok": true, "artifact": artifact, "expected_hash": expectedHash, "before": before, "after": after, "exact_bytes_and_mime": true}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile("/lab/frontend-proof.json", proof, 0600); err != nil {
		return err
	}
	return f.check()
}
