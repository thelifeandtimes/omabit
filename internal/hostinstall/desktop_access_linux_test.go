package hostinstall

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const desktopSSHConfig = "pubkeyauthentication yes\nforcecommand none\nchrootdirectory none\nusedns no\nauthenticationmethods any\nauthorizedkeysfile .ssh/authorized_keys .ssh/authorized_keys2\n"

func desktopAccessFixture(t *testing.T) (*fixture, ownerhost.Profile, DesktopAccessOptions, hostProbe, []byte) {
	t.Helper()
	f := newFixture(t)
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	if uid == 0 {
		uid, gid = 1000, 1000
	}
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: uid, GID: gid, Home: "/home/desktop", Piers: "/srv/urbit/piers"}
	f.mkdir(p.Home, 0700)
	f.mkdir(p.Home+"/.ssh", 0700)
	for _, path := range []string{p.Home, p.Home + "/.ssh"} {
		if err := os.Chown(f.i.path(path), int(uid), int(gid)); err != nil {
			t.Fatal(err)
		}
	}
	personal := append(hostPublic(9), []byte(" personal-admin\n# keep this comment\n")...)
	f.put(p.Home+"/.ssh/authorized_keys", string(personal), 0600)
	if err := os.Chown(f.i.path(p.Home+"/.ssh/authorized_keys"), int(uid), int(gid)); err != nil {
		t.Fatal(err)
	}
	old, _ := access.OwnerAuthorizedKey(hostPublic(2), "observer")
	f.put(access.DefaultKeys, old+"\n", 0644)
	f.put("/etc/urbit-runner/.setup.lock", "", 0600)
	f.put("/usr/bin/sshd", "inert native SSH fixture", 0755)
	f.i.run = func(_ context.Context, path string, args, env []string) ([]byte, error) {
		f.calls = append(f.calls, call{path, args, env})
		if path != "/usr/bin/sshd" || len(args) != 3 || args[0] != "-T" || args[1] != "-C" {
			t.Fatalf("pairing tried a mutation command: %s %v", path, args)
		}
		return []byte(desktopSSHConfig), nil
	}
	probe := func(context.Context, string) (protocol.Hello, bool, error) {
		return protocol.Hello{RunnerID: "runner-existing", Mode: "development-managed"}, true, nil
	}
	options := DesktopAccessOptions{RunnerID: "runner-existing", Role: "operator", SSHConnection: "100.64.0.1 43210 100.64.0.2 22"}
	return f, p, options, probe, personal
}

func TestDesktopAccessExistingRunnerRepeatAndRevocation(t *testing.T) {
	f, p, options, probe, personal := desktopAccessFixture(t)
	beforeKeys, _ := os.ReadFile(f.i.path(access.DefaultKeys))
	result, err := f.i.desktopAccess(context.Background(), p, hostPublic(1), options, probe)
	if err != nil || result.State != "authorized" || result.RunnerID != options.RunnerID || result.Port != 22 {
		t.Fatal(result, err)
	}
	line, _ := access.OwnerAuthorizedKey(hostPublic(1), "operator")
	path := f.i.path(p.Home + "/.ssh/authorized_keys")
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, append(bytes.Clone(personal), []byte(line+"\n")...)) {
		t.Fatal("changed administrative keys", string(got))
	}
	info, _ := os.Stat(path)
	if result, err = f.i.desktopAccess(context.Background(), p, hostPublic(1), options, probe); err != nil || result.State != "authorized" {
		t.Fatal(result, err)
	}
	again, _ := os.Stat(path)
	if !os.SameFile(info, again) || !info.ModTime().Equal(again.ModTime()) {
		t.Fatal("repeat replaced the SSH file")
	}
	options.Revoke = true
	offline := func(context.Context, string) (protocol.Hello, bool, error) {
		return protocol.Hello{}, false, errors.New("manager is stopped")
	}
	if result, err = f.i.desktopAccess(context.Background(), p, hostPublic(1), options, offline); err != nil || result.State != "revoked" {
		t.Fatal(result, err)
	}
	got, _ = os.ReadFile(path)
	keys, _ := os.ReadFile(f.i.path(access.DefaultKeys))
	if !bytes.Equal(got, personal) || !bytes.Equal(keys, beforeKeys) {
		t.Fatal("revocation changed another grant")
	}
	if _, err = f.i.desktopAccess(context.Background(), p, hostPublic(1), options, offline); err != nil {
		t.Fatal("repeat revoke", err)
	}
	for _, call := range f.calls {
		if call.path != "/usr/bin/sshd" {
			t.Fatal("restarted or installed something", call)
		}
	}
}

func TestDesktopAccessRefusesChangedIdentityPolicyAndUnsafeFiles(t *testing.T) {
	for _, fault := range []string{"runner", "forced-command", "mfa", "key-policy", "key-symlink", "ssh-symlink", "hard-link", "broad-permissions", "other-grant"} {
		t.Run(fault, func(t *testing.T) {
			f, p, options, probe, personal := desktopAccessFixture(t)
			path := f.i.path(p.Home + "/.ssh/authorized_keys")
			switch fault {
			case "runner":
				options.RunnerID = "runner-replacement"
			case "forced-command", "mfa", "key-policy":
				config := desktopSSHConfig
				if fault == "forced-command" {
					config = strings.ReplaceAll(config, "forcecommand none", "forcecommand internal-sftp")
				}
				if fault == "mfa" {
					config = strings.ReplaceAll(config, "authenticationmethods any", "authenticationmethods publickey,password")
				}
				if fault == "key-policy" {
					config = strings.ReplaceAll(config, "authorizedkeysfile .ssh/authorized_keys .ssh/authorized_keys2", "authorizedkeysfile /etc/central-keys/%u")
				}
				f.i.run = func(context.Context, string, []string, []string) ([]byte, error) { return []byte(config), nil }
			case "key-symlink":
				os.Remove(path)
				os.WriteFile(f.i.path("/outside"), personal, 0600)
				os.Symlink(f.i.path("/outside"), path)
			case "ssh-symlink":
				os.Rename(filepath.Dir(path), f.i.path("/outside"))
				os.Symlink(f.i.path("/outside"), filepath.Dir(path))
			case "hard-link":
				os.Link(path, f.i.path("/outside"))
			case "broad-permissions":
				os.Chmod(path, 0666)
			case "other-grant":
				os.WriteFile(path, append(hostPublic(1), []byte(" unrestricted\n")...), 0600)
			}
			before := tree(t, f.i.root)
			if _, err := f.i.desktopAccess(context.Background(), p, hostPublic(1), options, probe); err == nil {
				t.Fatal("accepted", fault)
			}
			if after := tree(t, f.i.root); !equalTree(before, after) {
				t.Fatal("refusal changed filesystem", fault)
			}
		})
	}
}

func equalTree(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestDesktopGrantMergeKeepsUnrelatedBytesAndRefusesRoleEscalation(t *testing.T) {
	line, _ := access.OwnerAuthorizedKey(hostPublic(1), "operator")
	observer, _ := access.OwnerAuthorizedKey(hostPublic(1), "observer")
	for _, central := range []bool{false, true} {
		if _, err := desktopKeyLines([]byte(observer+"\n"), line, central, false); err == nil {
			t.Fatal("silently changed key role")
		}
		if _, err := desktopKeyLines([]byte(line+"\n"+line+"\n"), line, central, false); err == nil {
			t.Fatal("accepted duplicate grant")
		}
	}
	before := []byte("# comment without final newline")
	got, err := desktopKeyLines(before, line, false, false)
	if err != nil || !bytes.HasPrefix(got, before) || bytes.Count(got, []byte(line)) != 1 {
		t.Fatal(string(got), err)
	}
	if _, err = desktopKeyLines([]byte("unparseable central authority"), line, true, false); err == nil {
		t.Fatal("accepted broken central grants")
	}
}

func TestDesktopSSHAcceptsNativeDirectiveCapitalization(t *testing.T) {
	p := ownerhost.Profile{Version: 1, Name: "owner", UID: 1000, GID: 1000, Home: "/home/owner", Piers: "/srv/urbit/piers"}
	config := "PubkeyAuthentication yes\nForceCommand none\nChrootDirectory none\nUseDNS no\nAuthenticationMethods any\nAuthorizedKeysFile .ssh/authorized_keys\n"
	if err := supportsDesktopKey(config, p); err != nil {
		t.Fatal(err)
	}
}
