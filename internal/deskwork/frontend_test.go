package deskwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"mime/multipart"
	"testing"
)

func frontendFixture(t *testing.T) (FrontendManifest, []SourceFile) {
	t.Helper()
	files := []SourceFile{{Path: "assets/logo.png", Bytes: []byte{0, 1, 0, 255}}, {Path: "index.html", Bytes: []byte("<!doctype html><title>Frozen</title>")}}
	entries := []File{}
	for _, f := range files {
		h := sha256.Sum256(f.Bytes)
		entries = append(entries, File{Path: f.Path, Bytes: int64(len(f.Bytes)), SHA256: hex.EncodeToString(h[:])})
	}
	source, err := NewManifest(entries)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewFrontendManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	return m, files
}

func TestFrontendMultipartPreservesCompleteArtifactAndTypes(t *testing.T) {
	m, files := frontendFixture(t)
	body, contentType, err := FrontendMultipart("example", m, files)
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatal(err)
	}
	r := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	part, err := r.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(part)
	if err != nil || part.FormName() != "desk" || string(data) != "example" {
		t.Fatal("desk changed", err)
	}
	for i, f := range files {
		part, err = r.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		_, disposition, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			t.Fatal(err)
		}
		data, err = io.ReadAll(part)
		if err != nil || part.FormName() != "glob" || disposition["filename"] != "bundle/"+f.Path || part.Header.Get("Content-Type") != m.Files[i].ContentType || !bytes.Equal(data, f.Bytes) {
			t.Fatal("frontend bytes or metadata changed", f.Path, err)
		}
	}
	if _, err = r.NextPart(); err != io.EOF {
		t.Fatal("extra multipart data", err)
	}
}

func TestFrontendRefusesChangedOrUnservableArtifacts(t *testing.T) {
	for _, scenario := range []string{"bytes", "trailing-zero", "missing", "extra", "order", "mime", "reserved", "index", "limit", "path"} {
		t.Run(scenario, func(t *testing.T) {
			m, files := frontendFixture(t)
			switch scenario {
			case "bytes":
				files[0].Bytes = []byte("changed")
			case "trailing-zero":
				files[0].Bytes = append(files[0].Bytes, 0)
				h := sha256.Sum256(files[0].Bytes)
				m.Files[0].File.Bytes = int64(len(files[0].Bytes))
				m.Files[0].File.SHA256 = hex.EncodeToString(h[:])
				m.Digest = frontendDigest(m.Files)
			case "missing":
				files = files[:1]
			case "extra":
				files = append(files, files[0])
			case "order":
				files[0], files[1] = files[1], files[0]
			case "mime":
				m.Files[0].ContentType = "text/html"
				m.Digest = frontendDigest(m.Files)
			case "reserved":
				m.Files[0].File.Path = "desk.js"
				m.Files[0].ContentType = frontendContentType("desk.js")
				m.Digest = frontendDigest(m.Files)
			case "index":
				m.Files = m.Files[:1]
				m.Digest = frontendDigest(m.Files)
			case "limit":
				m.Files[0].File.Bytes = MaxFrontendBytes
				m.Digest = frontendDigest(m.Files)
			case "path":
				m.Files[0].File.Path = "../index.html"
				m.Digest = frontendDigest(m.Files)
			}
			body, contentType, err := FrontendMultipart("example", m, files)
			if err == nil || body != nil || contentType != "" {
				t.Fatal("invalid upload prepared", scenario)
			}
		})
	}
}
