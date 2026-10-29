//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestLoginCodeDiscoversPrivatePortAndChecksIdentity(t *testing.T) {
	for _, variant := range []string{"default", "relocated", "wrong-identity", "changed-identity", "changed-before-code", "changed-after-code", "bad-code", "command-error", "wrong-container", "quarantined", "missing", "symlink", "hardlink", "fifo", "writable", "malformed"} {
		t.Run(variant, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			path := filepath.Join(managedPier(f, state), ".http.ports")
			port := "12321"
			if variant == "relocated" {
				port = "12322"
			}
			inventory := "8080 insecure public\n" + port + " insecure loopback\n"
			if err := os.WriteFile(path, []byte(inventory), 0600); err != nil {
				t.Fatal(err)
			}
			switch variant {
			case "missing":
				os.Remove(path)
			case "symlink":
				os.Rename(path, path+".other")
				os.Symlink(path+".other", path)
			case "hardlink":
				os.Link(path, path+".other")
			case "fifo":
				os.Remove(path)
				if err := syscall.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "writable":
				os.Chmod(path, 0666)
			case "malformed":
				os.WriteFile(path, []byte("PRIVATE invalid inventory"), 0600)
			}
			execute := f.d.execute
			var formulas []string
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, "--data-binary") {
					for _, required := range []string{"container exec " + id, "--disable", "--noproxy *", "--proxy ", "--max-filesize 1024", "http://127.0.0.1:" + port} {
						if !strings.Contains(joined, required) {
							t.Fatal("unsafe code command", required)
						}
					}
					if strings.Contains(joined, "conn.sock") || strings.Contains(joined, "lidlut") {
						t.Fatal("secret or control socket in command")
					}
					formula := "our"
					if strings.Contains(joined, `"dojo":"+code"`) {
						formula = "+code"
					}
					formulas = append(formulas, formula)
					if formula == "our" {
						if variant == "wrong-identity" || variant == "changed-identity" && len(formulas) == 3 {
							return `"~nec"`, nil
						}
						if variant == "changed-before-code" {
							os.WriteFile(path, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600)
						}
						return `"~zod\n"`, nil
					}
					if variant == "changed-after-code" {
						os.WriteFile(path, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600)
					}
					if variant == "bad-code" {
						return `{"code":"PRIVATE"}`, nil
					}
					if variant == "command-error" {
						return "PRIVATE", errors.New("PRIVATE")
					}
					return `"lidlut-tabwed-pillex-ridrup\n"`, nil
				}
				return execute(ctx, args, env)
			}
			selected := id
			if variant == "wrong-container" {
				selected = strings.Repeat("f", 64)
			}
			if variant == "quarantined" {
				state.MaintenanceUncertain = true
			}
			code, err := readLoginCode(context.Background(), state, selected, f.d)
			if variant == "default" || variant == "relocated" {
				if err != nil || code != "lidlut-tabwed-pillex-ridrup" || strings.Join(formulas, ",") != "our,+code,our" {
					t.Fatal("verified retrieval failed", err, formulas)
				}
			} else {
				stage, deadline := LoginCodeDiagnostic(err)
				expected := map[string]string{"wrong-identity": "identity-before", "changed-identity": "identity-after", "changed-before-code": "inventory-before-code", "changed-after-code": "inventory-after-code", "bad-code": "read-code", "command-error": "read-code", "wrong-container": "container", "quarantined": "selection"}[variant]
				if expected == "" {
					expected = "port-inventory"
				}
				if stage != expected || deadline {
					t.Fatal("incorrect nonsecret failure stage", stage, deadline)
				}
				if err == nil || code != "" || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("unsafe output accepted or leaked")
				}
				switch variant {
				case "wrong-identity", "changed-before-code":
					if strings.Join(formulas, ",") != "our" {
						t.Fatal("code read after failed identity/inventory", formulas)
					}
				case "changed-identity", "changed-after-code", "bad-code", "command-error":
				default:
					if len(formulas) != 0 {
						t.Fatal("refused selection reached administrative endpoint", formulas)
					}
				}
			}
		})
	}
}

func TestPrivateHTTPPortRejectsAmbiguousInventories(t *testing.T) {
	valid := "8080 insecure public\n12322 insecure loopback\n443 secure public\n"
	if port, err := privateHTTPPort(valid, 8080); err != nil || port != 12322 {
		t.Fatal(port, err)
	}
	for _, data := range []string{"", strings.Repeat("x", 1025), "12321 insecure loopback\n", "8081 insecure public\n12321 insecure loopback\n", valid + "12323 insecure loopback\n", strings.ReplaceAll(valid, "12322", "8080"), strings.ReplaceAll(valid, "12322", "012322"), strings.ReplaceAll(valid, "12322", "+12322"), strings.ReplaceAll(valid, "12322", "65536"), strings.ReplaceAll(valid, "12322", "80"), strings.ReplaceAll(valid, "loopback", "remote"), valid + "junk\n", valid + "\n"} {
		if _, err := privateHTTPPort(data, 8080); err == nil {
			t.Fatal("accepted malformed inventory")
		}
	}
}

func TestLoginCodeUsesRegisteredIdentity(t *testing.T) {
	for _, kind := range []string{"named-fake", "comet"} {
		t.Run(kind, func(t *testing.T) {
			fixture := managedFixture
			ship := "~nec"
			if kind == "comet" {
				fixture = managedCometFixture
				ship = fixtureComet
			}
			f, state, persist := fixture(t)
			if kind == "named-fake" {
				state.Fake = &ManagedFake{Ship: ship}
				f.d.hostIdentity = func(context.Context, string) (string, error) { return ship, nil }
			}
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			if err := os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("12322 insecure loopback\n8080 insecure public\n"), 0600); err != nil {
				t.Fatal(err)
			}
			execute := f.d.execute
			gotShip, codes := ship, 0
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, `"dojo":"our"`) {
					return `"` + gotShip + `"`, nil
				}
				if strings.Contains(joined, `"dojo":"+code"`) {
					codes++
					return `"lidlut-tabwed-pillex-ridrup"`, nil
				}
				return execute(ctx, args, env)
			}
			if code, err := readLoginCode(context.Background(), state, id, f.d); err != nil || code == "" || codes != 1 {
				t.Fatal("registered identity refused", err)
			}
			gotShip = "~zod"
			if code, err := readLoginCode(context.Background(), state, id, f.d); err == nil || code != "" || codes != 1 {
				t.Fatal("default identity accepted for nondefault registration")
			}
		})
	}
}
