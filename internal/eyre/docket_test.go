package eyre

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

type docketTransport func(*http.Request) (*http.Response, error)

func (f docketTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDocketUploadBoundToOwnerAndNeverReplayed(t *testing.T) {
	for _, variant := range []string{"ok", "lost-reply", "redirect", "too-large", "wrong-identity", "changed-artifact", "expired", "non-ack"} {
		t.Run(variant, func(t *testing.T) {
			data := []byte("<!doctype html><title>Fixture</title>\n")
			h := sha256.Sum256(data)
			source, err := deskwork.NewManifest([]deskwork.File{{Path: "index.html", Bytes: int64(len(data)), SHA256: hex.EncodeToString(h[:])}})
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := deskwork.NewFrontendManifest(source)
			if err != nil {
				t.Fatal(err)
			}
			files := []deskwork.SourceFile{{Path: "index.html", Bytes: data}}
			calls, uploads := 0, 0
			transport := docketTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Scheme+"://"+r.URL.Host != "http://127.0.0.1:18081" || r.GetBody != nil {
					t.Fatal("wrong origin or replayable request")
				}
				status, body := 200, "~zod"
				header := make(http.Header)
				switch r.URL.Path {
				case "/~/host":
					if r.Header.Get("Cookie") != "" {
						t.Fatal("unauthenticated identity carries session")
					}
					if variant == "wrong-identity" {
						body = "~nec"
					}
				case "/~/name":
					if r.Header.Get("Cookie") != "urbauth-~zod=fixture-secret" {
						t.Fatal("session absent")
					}
				case "/docket/upload":
					uploads++
					if r.Method != "POST" || r.Header.Get("Cookie") != "urbauth-~zod=fixture-secret" {
						t.Fatal("unauthenticated upload")
					}
					_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
					if err != nil {
						t.Fatal(err)
					}
					reader := multipart.NewReader(r.Body, params["boundary"])
					part, err := reader.NextPart()
					if err != nil {
						t.Fatal(err)
					}
					b, _ := io.ReadAll(part)
					if part.FormName() != "desk" || string(b) != "example" {
						t.Fatal("wrong desk")
					}
					part, err = reader.NextPart()
					if err != nil {
						t.Fatal(err)
					}
					b, _ = io.ReadAll(part)
					if !bytes.Equal(b, data) || part.Header.Get("Content-Type") != "text/html" {
						t.Fatal("asset changed")
					}
					body = "<p>successfully globbed</p>"
					switch variant {
					case "lost-reply":
						return nil, errors.New("PRIVATE fixture failure")
					case "redirect":
						status = 307
						header.Set("Location", "http://other.invalid/docket/upload")
					case "too-large":
						body = strings.Repeat("x", maxBody+1)
					case "non-ack":
						body = "not confirmed"
					}
				default:
					t.Fatal("unexpected route")
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			c, err := NewWithTransport("http://127.0.0.1:18081", "~zod", transport)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			c.session = session{Origin: c.origin, Ship: c.ship, Name: "urbauth-~zod", Value: "fixture-secret", Expires: time.Now().Add(time.Hour)}
			if variant == "changed-artifact" {
				files[0].Bytes = []byte("changed")
			}
			if variant == "expired" {
				c.session.Expires = time.Now().Add(-time.Hour)
			}
			err = c.UploadDocket(context.Background(), "example", artifact, files)
			if (variant == "ok") != (err == nil) || uploads > 1 || err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unsafe outcome", err, uploads)
			}
			if variant == "changed-artifact" && calls != 0 || (variant == "expired" || variant == "wrong-identity") && uploads != 0 {
				t.Fatal("request after failed preparation/identity/session")
			}
			if variant == "ok" && uploads != 1 {
				t.Fatal("missing upload")
			}
		})
	}
}
