package rootless

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Run the real hook wrapper and native entrypoint command with mock root identity,
// trust tools, timeout and verifier. All writes/executables stay in t.TempDir.
func TestNativeHookInvocationsAreBoundedInertAndNonFatal(t *testing.T) {
	for _, provider := range []string{"pacman", "deb"} {
		for _, fault := range []string{"", "verify", "timeout", "nonroot", "symlink", "writable", "bad-mode", "arguments"} {
			t.Run(provider+"/"+fault, func(t *testing.T) {
				root := t.TempDir()
				put := func(path, text string) {
					t.Helper()
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(text), 0755); err != nil {
						t.Fatal(err)
					}
				}
				files, err := PackageHookFiles(provider)
				if err != nil {
					t.Fatal(err)
				}
				wrapper := root + PackageRefreshPath
				text := strings.ReplaceAll(string(files[PackageRefreshPath]), "/usr", root+"/usr")
				text = strings.Replace(text, "directory in / ", "directory in '"+root+"' ", 1)
				put(wrapper, text)
				uid := "0"
				if fault == "nonroot" {
					uid = "1000"
				}
				put(root+"/usr/bin/id", "#!/bin/sh\n[ \"$1\" = -u ] || exit 2\nprintf '%s\\n' '"+uid+"'\n")
				mode := "755"
				if fault == "writable" {
					mode = "777"
				}
				if fault == "bad-mode" {
					mode = "invalid"
				}
				put(root+"/usr/bin/stat", "#!/bin/sh\n[ \"$1\" = -c ] && [ \"$3\" = -- ] || exit 2\ncase \"$2\" in %u) printf '0\\n';; %a) printf '%s\\n' '"+mode+"';; %h) printf '1\\n';; *) exit 2;; esac\n")
				// Assert -i and the exact fixed environment vector before dispatch.
				put(root+"/usr/bin/env", "#!/bin/sh\n[ \"$1\" = -i ] || exit 2\nshift\n[ \"$1\" = 'PATH="+root+"/usr/bin:"+root+"/usr/sbin:/bin:/sbin' ] && [ \"$2\" = HOME=/root ] && [ \"$3\" = LANG=C ] && [ \"$4\" = LC_ALL=C ] || exit 2\nshift 4\nexec \"$@\"\n")
				timeout := "exec \"$@\""
				if fault == "timeout" {
					timeout = "exit 124"
				}
				put(root+"/usr/bin/timeout", "#!/bin/sh\n[ \"$1\" = --kill-after=5s ] && [ \"$2\" = 130s ] || exit 2\nshift 2\n"+timeout+"\n")
				runner := root + "/usr/local/lib/urbit-runner/current/urbit-runner"
				code := "0"
				if fault == "verify" {
					code = "1"
				}
				put(runner, "#!/bin/sh\n[ \"$#\" = 4 ] && [ \"$1\" = verify-host-packages ] && [ \"$2\" = --acknowledge-local-verification ] && [ \"$3\" = --format ] && [ \"$4\" = json ] || exit 2\nif IFS= read -r ignored; then exit 2; fi\nprintf 'called\\n' > '"+root+"/verifier-called'\nprintf 'private verifier output\\n' >&2\nexit "+code+"\n")
				if fault == "symlink" {
					if err := os.Rename(runner, runner+".real"); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(runner+".real", runner); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var cmd *exec.Cmd
				if provider == "pacman" {
					hook := string(files[PacmanPackageHookPath])
					if !strings.Contains(hook, "When = PostTransaction\n") || strings.Contains(hook, "NeedsTargets") || strings.Contains(hook, "AbortOnFail") {
						t.Fatal("unsafe Pacman hook")
					}
					_, command, ok := strings.Cut(hook, "Exec = ")
					if !ok || strings.TrimSpace(command) != PackageRefreshPath {
						t.Fatal("unexpected hook command")
					}
					args := []string{wrapper}
					if fault == "arguments" {
						args = append(args, "--root", "../$(touch injected)")
					}
					cmd = exec.CommandContext(ctx, "/bin/sh", args...)
				} else {
					config := string(files[APTPackageHookPath])
					_, command, ok := strings.Cut(config, "DPkg::Post-Invoke { \"")
					command, end := strings.CutSuffix(command, "\"; };\n")
					if !ok || !end || command != PackageRefreshPath+" || true" {
						t.Fatal("unexpected APT hook command")
					}
					command = strings.ReplaceAll(command, PackageRefreshPath, wrapper)
					if fault == "arguments" {
						command = strings.Replace(command, " || true", " --root '../$(touch injected)' || true", 1)
					}
					cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
				}
				cmd.Dir = root
				cmd.Env = []string{"PATH=/untrusted", "HOME=/untrusted", "DOCKER_HOST=tcp://private:secret@host", "APT_CONFIG=/untrusted", "PACMAN_OPTS=--root ../untrusted"}
				cmd.Stdin = strings.NewReader("VERSION 3\n../bad.deb\n$(touch injected)\n; touch injected\n")
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("hook aborted OS transaction: %v %s", err, output)
				}
				if strings.Contains(string(output), "private verifier output") {
					t.Fatal("raw verifier output leaked")
				}
				if fault != "" && !strings.Contains(string(output), "authentication remains unresolved") {
					t.Fatal("missing explicit failure status", string(output))
				}
				if fault == "" && len(output) != 0 {
					t.Fatal("unexpected success output", string(output))
				}
				_, called := os.Stat(root + "/verifier-called")
				if (called == nil) != (fault == "" || fault == "verify") {
					t.Fatal("wrong verifier dispatch", fault, called)
				}
				if _, err := os.Stat(root + "/injected"); !os.IsNotExist(err) {
					t.Fatal("transaction input executed")
				}
			})
		}
	}
}
