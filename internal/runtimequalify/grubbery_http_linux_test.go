//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestGrubberyFixedCurlPreservesBinaryStdinQueryAndCookie(t *testing.T) {
	body := []byte{0, 255, 1, 0, '$', 'H', 'O', 'M', 'E', '\n'}
	cookie := "urbauth-~zod=fixture-$HOME-`uname`"
	received := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(got, body) || r.Method != "PUT" || r.URL.RawQuery != "blot=%2Fhoon" || r.Header.Get("Cookie") != cookie || r.Header.Get("Content-Type") != "application/octet-stream" {
			received <- errors.New("request changed")
			w.WriteHeader(400)
			return
		}
		received <- nil
		w.WriteHeader(201)
		_, _ = w.Write([]byte("Created"))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := server.URL + "/grubbery/api/make/owned/raw.bin?blot=%2Fhoon"
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", grubberyHTTPScript, "omabit-private-app-http", "PUT", url)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	cmd.Stdin = bytes.NewReader(append([]byte("application/octet-stream\n"+cookie+"\n"), body...))
	for _, arg := range cmd.Args {
		if strings.Contains(arg, cookie) {
			t.Fatal("cookie entered argv")
		}
	}
	raw, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := <-received; err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("PUT", url, nil)
	response, err := decodeDeskHTTP(string(raw), req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	reply, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 201 || string(reply) != "Created" {
		t.Fatal("native HTTP response changed", err)
	}
}
func TestGrubberyTransportRefusesChangedSelectionBeforeRuntime(t *testing.T) {
	cases := []string{"origin", "host", "method", "path", "query", "fragment", "raw-path", "body", "content-type", "extra-header", "multiple-cookie", "transfer-encoding"}
	for _, variant := range cases {
		t.Run(variant, func(t *testing.T) {
			checks := 0
			runtime := &DeskRuntime{q: &qualifier{}, httpEndpoint: "http://127.0.0.1:8080", check: func(context.Context) error { checks++; return errors.New("should never reach runtime") }}
			selected := grubbery.Request{Method: "PUT", Path: "/grubbery/api/make/owned/nex/state.hoon", Query: "blot=%2Fhoon", ContentType: "text/x-hoon", Body: []byte(":: exact\n")}
			req, _ := http.NewRequest("PUT", runtime.httpEndpoint+selected.Path+"?"+selected.Query, bytes.NewReader(selected.Body))
			req.Header.Set("Content-Type", selected.ContentType)
			req.Header.Set("Cookie", "fixture")
			switch variant {
			case "origin":
				req.URL.Host = "other.invalid"
			case "host":
				req.Host = "other.invalid"
			case "method":
				req.Method = "DELETE"
			case "path":
				req.URL.Path = "/grubbery/api/cull/owned"
			case "query":
				req.URL.RawQuery = "force=1"
			case "fragment":
				req.URL.Fragment = "other"
			case "raw-path":
				req.URL.RawPath = "/encoded"
			case "body":
				req.Body = io.NopCloser(strings.NewReader(":: changed"))
			case "content-type":
				req.Header.Set("Content-Type", "application/json")
			case "extra-header":
				req.Header.Set("Authorization", "private")
			case "multiple-cookie":
				req.Header.Add("Cookie", "second")
			case "transfer-encoding":
				req.TransferEncoding = []string{"chunked"}
			}
			if _, err := (grubberyHTTPTransport{runtime, selected}).RoundTrip(req); err == nil {
				t.Fatal("changed selection accepted")
			}
			if checks != 0 {
				t.Fatal("invalid selection probed runtime")
			}
		})
	}
}
