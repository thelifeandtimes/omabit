package onboarding

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func remoteFixture(t *testing.T) (string, string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("desktop onboarding runs as a non-root user")
	}
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	helper := filepath.Join(dir, "urbit-runner")
	if err := os.WriteFile(helper, []byte("inert fixture helper; never executed"), 0700); err != nil {
		t.Fatal(err)
	}
	return helper, filepath.Join(dir, "collector")
}

func TestRemotePairingRetainsKeyAndTargetAcrossInterruptedRetry(t *testing.T) {
	helper, config := remoteFixture(t)
	options := RemoteOptions{Host: "sourcebox", Name: "sourcebox"}
	status, err := prepareRemote(context.Background(), options, config, "", false, helper)
	if err != nil || status.ID == "" || status.Fingerprint == "" {
		t.Fatal(status, err)
	}
	root, _ := remoteRoot()
	job, dir, err := readRemote(root, status.ID)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := os.ReadFile(filepath.Join(dir, "id_ed25519"))
	if _, err = prepareRemote(context.Background(), options, config, "", false, helper); err == nil {
		t.Fatal("opened a duplicate pending setup")
	}
	job.Phase = "authorize"
	job.RunnerID = "runner-pinned"
	job.Mode = "development-managed"
	saveRemote(dir, &job)
	unlock, err := lockSetup(dir)
	if err != nil {
		t.Fatal(err)
	}
	state, err := RemoteState(status.ID)
	if err != nil || !state.Busy {
		t.Fatal(state, err)
	}
	unlock()
	state, err = RemoteState(status.ID)
	if err != nil || state.Busy || state.Phase != "interrupted" {
		t.Fatal(state, err)
	}
	state, err = prepareRemote(context.Background(), options, config, status.ID, false, helper)
	if err != nil || state.ID != status.ID || state.Fingerprint != status.Fingerprint || state.RunnerID != "runner-pinned" {
		t.Fatal(state, err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "id_ed25519"))
	if string(after) != string(key) {
		t.Fatal("retry rotated a key")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(data), "PRIVATE KEY") {
		t.Fatal("state leaked a private key")
	}
}

func TestRemotePairingRefusesProfileAndInstallerDrift(t *testing.T) {
	helper, config := remoteFixture(t)
	status, err := prepareRemote(context.Background(), RemoteOptions{Host: "host"}, config, "", false, helper)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := remoteRoot()
	job, dir, _ := readRemote(root, status.ID)
	job.Phase = "failed"
	saveRemote(dir, &job)
	if _, err = prepareRemote(context.Background(), RemoteOptions{}, config+"-other", status.ID, false, helper); err == nil {
		t.Fatal("reused another profile")
	}
	os.WriteFile(helper, []byte("another binary"), 0700)
	if _, err = prepareRemote(context.Background(), RemoteOptions{}, config, status.ID, false, helper); err != nil {
		t.Fatal("original installer was not retained", err)
	}
	again, _, _ := readRemote(root, status.ID)
	if again.HelperSHA256 != job.HelperSHA256 {
		t.Fatal("silently replaced an interrupted installer")
	}
	again.Phase = "failed"
	saveRemote(dir, &again)
	os.Chmod(again.Helper, 0600)
	os.WriteFile(again.Helper, []byte("tampered installer"), 0600)
	if _, err = prepareRemote(context.Background(), RemoteOptions{}, config, status.ID, false, helper); err == nil {
		t.Fatal("accepted tampered retained installer")
	}
	if _, err = RemoteState(strings.Repeat("b", 32)); err == nil {
		t.Fatal("accepted unknown pairing")
	}
	if _, err = os.Stat(filepath.Join(root, strings.Repeat("b", 32))); !os.IsNotExist(err) {
		t.Fatal("status created an unknown pairing")
	}
	os.Chmod(filepath.Join(dir, ".lock"), 0666)
	if _, err = RemoteState(status.ID); err == nil {
		t.Fatal("unsafe lock masqueraded as busy")
	}
}

func TestRemoteStatusPendingGraceAndUnknownFormats(t *testing.T) {
	helper, config := remoteFixture(t)
	status, err := prepareRemote(context.Background(), RemoteOptions{Host: "host"}, config, "", false, helper)
	if err != nil {
		t.Fatal(err)
	}
	state, err := RemoteState("")
	if err != nil || !state.Busy {
		t.Fatal(state, err)
	}
	root, _ := remoteRoot()
	job, dir, _ := readRemote(root, status.ID)
	job.UpdatedAt = time.Now().Add(-time.Minute)
	data, _ := json.Marshal(job)
	writePrivate(filepath.Join(dir, "state.json"), data)
	state, err = RemoteState(status.ID)
	if err != nil || state.Busy || state.Phase != "interrupted" {
		t.Fatal(state, err)
	}
	job.Version = 99
	data, _ = json.Marshal(job)
	writePrivate(filepath.Join(dir, "state.json"), data)
	if _, err = RemoteState(status.ID); err == nil {
		t.Fatal("accepted unknown state format")
	}
}

func TestManagementConfigUsesDedicatedIdentityAndPreservesConfiguredRoute(t *testing.T) {
	_, _ = remoteFixture(t)
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	job := remoteJob{RemoteStatus: RemoteStatus{ID: strings.Repeat("a", 32), Options: RemoteOptions{Host: "sourcebox", SSHConfig: "/private/original config"}}}
	r := sshResolved{Host: "sourcebox.example", User: "admin", Port: "2222", HostKeyAlias: "pinned-sourcebox", ProxyJump: "bastion"}
	config, err := managementConfig(dir, job, r, "owner")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(config.ConfigFile)
	text := string(data)
	for _, expected := range []string{"User owner\n", "Port 2222\n", "IdentityAgent none\n", "IdentitiesOnly yes\n", "BatchMode yes\n", "StrictHostKeyChecking yes\n", "ControlPath none\n", "ProxyJump bastion\n", "Include \"/private/original config\"", "HostKeyAlias pinned-sourcebox\n"} {
		if !strings.Contains(text, expected) {
			t.Fatal("missing", expected, text)
		}
	}
	if config.Host == job.Options.Host || strings.Contains(text, "id_rsa") {
		t.Fatal("reused personal credentials")
	}
	r.Host = "host\nInclude /unrelated"
	if _, err = managementConfig(dir, job, r, "owner"); err == nil {
		t.Fatal("accepted SSH config injection")
	}
	for _, host := range []string{"-oProxyCommand=bad", "host;command", "host\nother", "user@host"} {
		o := RemoteOptions{Host: host}
		if o.validate() == nil {
			t.Fatal("accepted", host)
		}
	}
	args := strings.Join(sshBase(RemoteOptions{Host: "sourcebox"}), " ")
	for _, option := range []string{"-oClearAllForwardings=yes", "-oForwardAgent=no", "-oPermitLocalCommand=no", "-oRemoteCommand=none"} {
		if !strings.Contains(args, option) {
			t.Fatal("bootstrap enabled incidental SSH behavior", option)
		}
	}
}

func TestFailedRevocationCannotResumeAsAuthorization(t *testing.T) {
	helper, config := remoteFixture(t)
	status, err := prepareRemote(context.Background(), RemoteOptions{Host: "host"}, config, "", false, helper)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := remoteRoot()
	job, dir, _ := readRemote(root, status.ID)
	job.RunnerID = "runner-original"
	job.Mode = "development-managed"
	job.Revoke = true
	job.Phase = "failed"
	if err = saveRemote(dir, &job); err != nil {
		t.Fatal(err)
	}
	again, err := prepareRemote(context.Background(), RemoteOptions{}, config, status.ID, false, helper)
	if err != nil || !again.Revoking {
		t.Fatal("retry changed revocation into authorization", again, err)
	}
}

func TestRemoteForgetIsIdempotentAndPreservesReboundCredentials(t *testing.T) {
	_, config := remoteFixture(t)
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(filepath.Dir(config), "pairing")
	job := remoteJob{RemoteStatus: RemoteStatus{ID: strings.Repeat("a", 32), RunnerID: "runner-pinned", Connection: "original-name"}, ConfigDir: config}
	entry := collection.Connection{Name: "renamed", RunnerID: job.RunnerID, Mode: "development-managed", Revision: 1, EnrolledAt: time.Now().UTC(), Transport: "ssh", SSH: &client.SSHConfig{Host: "omabit-" + job.ID, ConfigFile: filepath.Join(dir, "ssh_config")}}
	put := func(entry collection.Connection) {
		data, _ := json.Marshal(map[string]any{"version": 1, "connections": []collection.Connection{entry}})
		if e := os.WriteFile(filepath.Join(config, "runners.json"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	put(entry)
	if err = forgetRemote(context.Background(), job, dir); err != nil {
		t.Fatal(err)
	}
	if err = forgetRemote(context.Background(), job, dir); err != nil {
		t.Fatal("repeat forget", err)
	}
	rows, err := r.List(context.Background())
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	entry.SSH = &client.SSHConfig{Host: "another-key", ConfigFile: filepath.Join(dir, "other-config")}
	put(entry)
	if err = forgetRemote(context.Background(), job, dir); err != nil {
		t.Fatal(err)
	}
	rows, err = r.List(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal("removed an unrelated credential", rows, err)
	}
}

func TestPendingPairingBindsResolvedEndpointAndPinsSessionArguments(t *testing.T) {
	job := remoteJob{RemoteStatus: RemoteStatus{Options: RemoteOptions{Host: "alias"}}}
	endpoint := sshResolved{Host: "machine.example", User: "admin", Port: "2200", KnownHosts: []string{"/private/known_hosts"}}
	if err := bindRemoteEndpoint(&job, endpoint); err != nil {
		t.Fatal(err)
	}
	saved := job.EndpointSHA256
	endpoint.KnownHosts = []string{"/another/trust-file"}
	if err := bindRemoteEndpoint(&job, endpoint); err != nil || job.EndpointSHA256 != saved {
		t.Fatal("trust-source changes moved the target", err)
	}
	for _, change := range []func(*sshResolved){func(r *sshResolved) { r.Host = "replacement" }, func(r *sshResolved) { r.User = "other" }, func(r *sshResolved) { r.Port = "22" }, func(r *sshResolved) { r.ProxyJump = "different-bastion" }} {
		other := endpoint
		change(&other)
		if bindRemoteEndpoint(&job, other) == nil {
			t.Fatal("pending target changed", other)
		}
	}
	args := strings.Join(sessionSSHBase(job), " ")
	for _, expected := range []string{"-oHostName=machine.example", "-l admin", "-p 2200", "-oProxyCommand=none"} {
		if !strings.Contains(args, expected) {
			t.Fatal("session depended on a mutable alias", args)
		}
	}
}

func TestBootstrapArgumentsAreAcceptedByRealOpenSSHWithoutConnecting(t *testing.T) {
	if _, err := os.Stat("/usr/bin/ssh"); err != nil {
		t.Skip("OpenSSH unavailable")
	}
	job := remoteJob{RemoteStatus: RemoteStatus{Options: RemoteOptions{Host: "fixture.invalid", SSHConfig: "/dev/null"}}}
	if err := bindRemoteEndpoint(&job, sshResolved{Host: "fixture.invalid", User: "fixture", Port: "22"}); err != nil {
		t.Fatal(err)
	}
	args := append([]string{"-G"}, masterSSHArgs(job, "/tmp/omabit-fixture-control", "/tmp/omabit-fixture-known")...)
	output, err := exec.Command("/usr/bin/ssh", args...).CombinedOutput()
	if err != nil {
		t.Fatal(string(output), err)
	}
	text := strings.ToLower(string(output))
	for _, value := range []string{"sessiontype none", "batchmode no", "clearallforwardings yes", "controlpath /tmp/omabit-fixture-control"} {
		if !strings.Contains(text, value) {
			t.Fatal("wrong bootstrap options", value, text)
		}
	}
}
