//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gatewayProgramFixture(t *testing.T) (string, string, *RuntimeInputVerifier) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "sealed")
	program := filepath.Join(root, "gateway", "docker_gateway.cjs")
	mustInput(t, os.MkdirAll(filepath.Dir(program), 0700))
	raw := []byte("synthetic initializer, never executed\n")
	mustInput(t, os.WriteFile(program, raw, 0400))
	mustInput(t, os.Chmod(filepath.Dir(program), 0500))
	mustInput(t, os.Chmod(root, 0500))
	v, err := OpenRuntimeInputVerifier(context.Background(), root, strings.Repeat("a", 64), strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		v.Close()
		filepath.Walk(base, func(p string, i os.FileInfo, err error) error {
			if err == nil && i.IsDir() {
				os.Chmod(p, 0700)
			}
			return nil
		})
	})
	return root, inputSHA(raw), v
}
func TestRuntimeInputGatewayProgramPin(t *testing.T) {
	root, pin, v := gatewayProgramFixture(t)
	before, err := os.ReadFile(filepath.Join(root, "gateway/docker_gateway.cjs"))
	mustInput(t, err)
	for range 2 {
		mustInput(t, v.VerifyGatewayInitializer(context.Background(), pin))
	}
	if _, err := v.Verify(context.Background()); err == nil {
		t.Fatal("single program pin qualified an incomplete bundle")
	}
	after, err := os.ReadFile(filepath.Join(root, "gateway/docker_gateway.cjs"))
	mustInput(t, err)
	if string(before) != string(after) {
		t.Fatal("changed program")
	}
	mustInput(t, v.Close())
	if err := v.VerifyGatewayInitializer(context.Background(), pin); err == nil {
		t.Fatal("closed program handle accepted")
	}
}
func TestRuntimeInputGatewayProgramRefusals(t *testing.T) {
	for _, kind := range []string{"wrong-pin", "invalid-pin", "changed", "missing", "symlink", "hardlink", "public", "executable", "oversize", "root-replacement"} {
		t.Run(kind, func(t *testing.T) {
			root, pin, v := gatewayProgramFixture(t)
			program := filepath.Join(root, "gateway/docker_gateway.cjs")
			switch kind {
			case "wrong-pin":
				pin = strings.Repeat("c", 64)
			case "invalid-pin":
				pin = "bad"
			case "changed":
				mustInput(t, os.Chmod(program, 0600))
				mustInput(t, os.WriteFile(program, []byte("changed"), 0600))
				mustInput(t, os.Chmod(program, 0400))
			case "missing":
				mustInput(t, os.Chmod(filepath.Dir(program), 0700))
				mustInput(t, os.Remove(program))
				mustInput(t, os.Chmod(filepath.Dir(program), 0500))
			case "symlink":
				mustInput(t, os.Chmod(filepath.Dir(program), 0700))
				mustInput(t, os.Rename(program, program+".retained"))
				mustInput(t, os.Symlink("docker_gateway.cjs.retained", program))
				mustInput(t, os.Chmod(filepath.Dir(program), 0500))
			case "hardlink":
				mustInput(t, os.Link(program, filepath.Join(filepath.Dir(root), "alias")))
			case "public":
				mustInput(t, os.Chmod(program, 0444))
			case "executable":
				mustInput(t, os.Chmod(program, 0500))
			case "oversize":
				mustInput(t, os.Chmod(program, 0600))
				mustInput(t, os.Truncate(program, (1<<20)+1))
				mustInput(t, os.Chmod(program, 0400))
			case "root-replacement":
				mustInput(t, os.Rename(root, root+".retained"))
				mustInput(t, os.MkdirAll(filepath.Dir(program), 0700))
				raw, err := os.ReadFile(filepath.Join(root+".retained", "gateway/docker_gateway.cjs"))
				mustInput(t, err)
				mustInput(t, os.WriteFile(program, raw, 0400))
				mustInput(t, os.Chmod(filepath.Dir(program), 0500))
				mustInput(t, os.Chmod(root, 0500))
			}
			if err := v.VerifyGatewayInitializer(context.Background(), pin); err == nil {
				t.Fatal("accepted mismatched program")
			}
		})
	}
	_, pin, v := gatewayProgramFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := v.VerifyGatewayInitializer(ctx, pin); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
