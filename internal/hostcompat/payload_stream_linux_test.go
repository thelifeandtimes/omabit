//go:build linux

package hostcompat

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

type smallPayloadReader struct{ io.Reader }

func (r smallPayloadReader) Read(p []byte) (int, error) {
	if len(p) > 7 {
		p = p[:7]
	}
	return r.Reader.Read(p)
}

func TestPackagePayloadStreamsBothProviders(t *testing.T) {
	data := makeTar(t, map[string][]byte{"usr/bin/dockerd": []byte("engine")})
	for _, provider := range []string{"pacman", "deb"} {
		t.Run(provider, func(t *testing.T) {
			calls := 0
			v := verifier{run: func(context.Context, string, []string, int64) ([]byte, error) {
				t.Fatal("buffered fallback used")
				return nil, nil
			}, stream: func(_ context.Context, tool string, args []string, limit int64, consume func(io.Reader) error) error {
				calls++
				wantTool := "/usr/bin/bsdtar"
				wantArgs := []string{"-cf", "-", "--format=pax", "--", "@/package"}
				if provider == "deb" {
					wantTool = "/usr/bin/dpkg-deb"
					wantArgs = []string{"--fsys-tarfile", "/package"}
				}
				if tool != wantTool || !reflect.DeepEqual(args, wantArgs) || limit != maxArtifact {
					t.Fatalf("decoder contract changed: %s %v %d", tool, args, limit)
				}
				return consume(smallPayloadReader{bytes.NewReader(data)})
			}}
			got, err := v.packagePayload(context.Background(), provider, "package", []string{"usr/bin/dockerd"})
			if err != nil || calls != 1 || got["usr/bin/dockerd"] != fmt.Sprintf("%x", sha256.Sum256([]byte("engine"))) {
				t.Fatalf("streamed payload: %v %v", got, err)
			}
		})
	}
}

func TestPackagePayloadStreamFailureNeverFallsBack(t *testing.T) {
	valid := makeTar(t, map[string][]byte{"usr/bin/dockerd": []byte("engine")})
	var duplicate bytes.Buffer
	tw := tar.NewWriter(&duplicate)
	for i := 0; i < 2; i++ {
		if err := tw.WriteHeader(&tar.Header{Name: "usr/bin/dockerd", Mode: 0755, Size: 1}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte("x"))
	}
	tw.Close()
	var link bytes.Buffer
	tw = tar.NewWriter(&link)
	tw.WriteHeader(&tar.Header{Name: "usr/bin/dockerd", Typeflag: tar.TypeLink, Linkname: "elsewhere"})
	tw.Close()
	cases := []struct {
		name string
		data []byte
		late bool
	}{
		{"decoder-nonzero-after-valid-payload", valid, true},
		{"truncated", valid[:514], false},
		{"duplicate", duplicate.Bytes(), false},
		{"linked-payload", link.Bytes(), false},
		{"missing", makeTar(t, map[string][]byte{"different": []byte("x")}), false},
		{"traversal", makeTar(t, map[string][]byte{"../escape": []byte("x")}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := verifier{run: func(context.Context, string, []string, int64) ([]byte, error) {
				t.Fatal("failed stream retried")
				return nil, nil
			}, stream: func(_ context.Context, _ string, _ []string, _ int64, consume func(io.Reader) error) error {
				err := consume(smallPayloadReader{bytes.NewReader(tc.data)})
				if tc.late {
					return errors.New("decoder failed after payload")
				}
				return err
			}}
			got, err := v.packagePayload(context.Background(), "pacman", "package", []string{"usr/bin/dockerd"})
			if err == nil || got != nil {
				t.Fatalf("partial evidence accepted: %v %v", got, err)
			}
		})
	}
}

func TestPackagePayloadStreamChargesWholeOutput(t *testing.T) {
	data := append(makeTar(t, map[string][]byte{"usr/bin/dockerd": []byte("engine")}), bytes.Repeat([]byte{0}, 4096)...)
	w := &evidenceWork{limits: workLimits{commands: int64(len(data) - 1)}}
	run := func(_ context.Context, _ string, _ []string, limit int64, consume func(io.Reader) error) error {
		r := io.LimitReader(bytes.NewReader(data), limit+1)
		err := consume(r)
		if err == nil {
			_, err = io.Copy(io.Discard, r)
		}
		return err
	}
	v := verifier{stream: boundStreams(run, w)}
	got, err := v.packagePayload(context.Background(), "pacman", "package", []string{"usr/bin/dockerd"})
	if !errors.Is(err, errBudget) || got != nil {
		t.Fatalf("trailing output escaped aggregate budget: %v %v %+v", got, err, w)
	}
}

func TestActualPacmanStreamingAuthenticatesAndReusesPayload(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	s := inspect(m, "linux", "amd64")
	native := fixtureRunner(t, m, home)
	streams := 0
	v := verifier{fsys: m, run: func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		if tool == "/usr/bin/bsdtar" && args[0] == "-cf" {
			t.Fatal("production payload buffered")
		}
		return native(ctx, tool, args, limit)
	}, stream: func(ctx context.Context, tool string, args []string, limit int64, consume func(io.Reader) error) error {
		streams++
		b, e := native(ctx, tool, args, limit)
		if e != nil {
			return e
		}
		return consume(smallPayloadReader{bytes.NewReader(b)})
	}}
	for _, p := range s.report.Packages[:2] {
		r, e := v.verifyPackage(context.Background(), "pacman", p, s.owners[p.Role])
		if e != nil || !r.Authorized {
			t.Fatalf("signed %s payload failed: %v", p.Role, e)
		}
	}
	if streams != 1 {
		t.Fatalf("shared archive decoded %d times", streams)
	}
	m["usr/bin/dockerd"].Data = []byte("tampered")
	if _, e := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"]); e == nil {
		t.Fatal("modified installed bytes authenticated")
	}
}
