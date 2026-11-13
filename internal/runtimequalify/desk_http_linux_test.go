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
)

func TestDeskHTTPFixedScriptPreservesStdinAndChunkedResponse(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			body := []byte{}
			if method == "POST" {
				body = append(bytes.Repeat([]byte{0, 1, 255, 42}, 16384), 0)
			}
			cookie := "urbauth-~zod=fixture-$HOME-`uname`"
			requestResult := make(chan error, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil || !bytes.Equal(got, body) || r.Method != method || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Cookie") != cookie {
					requestResult <- errors.New("headers, method or binary body changed")
					w.WriteHeader(400)
					return
				}
				requestResult <- nil
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Set-Cookie", "fixture=response-secret")
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				_, _ = w.Write([]byte{0, 255, 42, 0})
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", deskHTTPScript, "omabit-private-http", method, server.URL+"/docket/upload")
			cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
			cmd.Stdin = bytes.NewReader(append([]byte("application/octet-stream\n"+cookie+"\n"), body...))
			for _, arg := range cmd.Args {
				if strings.Contains(arg, cookie) {
					t.Fatal("credential in argv")
				}
			}
			out, err := cmd.Output()
			if err != nil {
				t.Fatal("fixed curl script failed", err)
			}
			if requestErr := <-requestResult; requestErr != nil {
				t.Fatal(requestErr)
			}
			req, _ := http.NewRequest(method, server.URL+"/docket/upload", nil)
			response, err := decodeDeskHTTP(string(out), req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 200 || !bytes.Equal(data, []byte{0, 255, 42, 0}) || response.Header.Get("Set-Cookie") != "fixture=response-secret" {
				t.Fatal("response changed", err)
			}
		})
	}
}

func TestDeskHTTPResponseRejectsTruncationAndExtraData(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://127.0.0.1:8080/docket/upload", nil)
	for _, raw := range []string{
		"", "not HTTP", "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nx",
		"HTTP/1.1 200 OK\r\nContent-Length: 1\r\n\r\nxy",
		"HTTP/1.1 101 Switching Protocols\r\n\r\n",
		"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n2\r\nx\r\n0\r\n\r\n",
	} {
		if _, err := decodeDeskHTTP(raw, req); err == nil {
			t.Fatal("invalid response accepted")
		}
	}
	for _, raw := range []string{
		"HTTP/1.1 100 Continue\r\n\r\nHTTP/1.1 200 OK\r\nContent-Length: 1\r\n\r\nx",
		"HTTP/1.1 303 See Other\r\nLocation: /\r\nContent-Length: 0\r\n\r\n",
	} {
		r, err := decodeDeskHTTP(raw, req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
	}
}

func TestDeskHTTPTransportRejectsUnselectedRequests(t *testing.T) {
	for _, variant := range []string{"origin", "route", "query", "method", "header", "header-injection", "body", "host", "rawpath"} {
		t.Run(variant, func(t *testing.T) {
			checks := 0
			s := &DeskRuntime{httpEndpoint: "http://127.0.0.1:8080", check: func(context.Context) error { checks++; return errors.New("fixture should not reach runtime") }}
			r, _ := http.NewRequest("GET", "http://127.0.0.1:8080/~/host", nil)
			switch variant {
			case "origin":
				r.URL.Host = "127.0.0.1:8081"
			case "route":
				r.URL.Path = "/~/channel/arbitrary"
			case "query":
				r.URL.RawQuery = "changed=1"
			case "method":
				r.Method = "DELETE"
			case "header":
				r.Header.Set("Authorization", "PRIVATE")
			case "header-injection":
				r.Header.Set("Cookie", "PRIVATE\r\nX-Evil: yes")
			case "body":
				r.Body = io.NopCloser(strings.NewReader("PRIVATE"))
				r.ContentLength = 7
			case "host":
				r.Host = "other.invalid"
			case "rawpath":
				r.URL.RawPath = "/%7e/host"
			}
			_, err := (deskHTTPTransport{s}).RoundTrip(r)
			if err == nil || checks != 0 || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unselected request reached runtime", err, checks)
			}
		})
	}
}
