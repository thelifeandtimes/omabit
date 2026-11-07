//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func (f *nativeDeskFixture) recordEffect(id int32, state string) error {
	file, err := os.OpenFile("/lab/workflow-effects.jsonl", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
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
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err = w.WriteField("desk", f.Desk); err != nil {
		return err
	}
	part, err := w.CreatePart(textproto.MIMEHeader{"Content-Disposition": {`form-data; name="glob"; filename="dist/index.html"`}, "Content-Type": {"text/html"}})
	if err != nil {
		return err
	}
	if _, err = part.Write(content); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:18081/docket/upload", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: session.Name, Value: session.Value})
	req.GetBody = nil
	id := f.effects.Add(1)
	if err = f.recordEffect(id, "dispatched"); err != nil {
		return err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return errors.New("fixture upload response unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 || !strings.Contains(string(data), "successfully globbed") {
		return errors.New("fixture upload not acknowledged")
	}
	if err = f.check(); err != nil {
		return err
	}
	if err = f.recordEffect(id, "acknowledged"); err != nil {
		return err
	}
	// Verify actual served bytes, independently of metadata or HTTP upload success.
	req, err = http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:18081/apps/"+f.Desk+"/index.html", nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: session.Name, Value: session.Value})
	resp, err = f.client.Do(req)
	if err != nil {
		return errors.New("fixture frontend read unavailable")
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 || !bytes.Equal(data, content) {
		return errors.New("fixture served frontend differs from frozen content")
	}
	return f.check()
}
