package hostinstall

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

// This is a management-only compatibility bridge, never a fresh installation.
// It does not stop a legacy manager: the operator must explicitly stop its unit
// after work has settled. The runtime's own flock closes the restart race.
func (i *installer) installOwnerOffline(ctx context.Context, bundle string, public, binary []byte, p ownerhost.Profile, progress func(string)) (result HostResult, err error) {
	defer func() { result.Partial = err != nil && i.mutated }()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return result, errors.New("offline upgrade requires OS authorization on Linux amd64")
	}
	if managed.StateSchemaVersion < 20 {
		return result, errors.New("offline upgrade requires an integrated schema20+ installer")
	}
	if err := p.Validate(); err != nil {
		return result, err
	}
	if len(binary) == 0 || len(binary) > 128<<20 {
		return result, errors.New("invalid authorized installer executable")
	}
	profile, err := i.readInstalled(ctx, ownerhost.Path, 8192)
	var installed ownerhost.Profile
	if err != nil || json.Unmarshal(profile, &installed) != nil || installed != p {
		return result, errors.New("offline upgrade requires the unchanged installed owner profile; no account/path migration or fresh installation")
	}
	unit, err := i.readInstalled(ctx, managerUnit, maxOutput)
	if err != nil || !bytes.Equal(unit, ownerManagerUnit(p)) {
		return result, errors.New("installed owner management unit differs; preserve administrator changes and restore its reviewed invocation before upgrading")
	}
	for _, path := range []string{managerUnit + ".d", filepath.Join(p.Home, ".config/systemd/user/urbit-runner.service"), filepath.Join(p.Home, ".config/systemd/user/urbit-runner.service.d")} {
		if _, err := i.lstat(i.path(path)); !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("offline activation refuses unreviewed management override %s", path)
		}
	}
	if _, err := i.verified(bundle); err != nil {
		return result, err
	}
	line, err := access.OwnerAuthorizedKey(public, "operator")
	if err != nil {
		return result, err
	}
	grant, err := access.ParseLine(line)
	if err != nil {
		return result, err
	}
	keys, err := i.readInstalled(ctx, access.DefaultKeys, 65536)
	if err != nil {
		return result, err
	}
	found := false
	for _, raw := range strings.Split(string(keys), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		g, err := access.ParseLine(raw)
		if err != nil {
			return result, err
		}
		found = found || g.Fingerprint == grant.Fingerprint && g.Role == "operator"
	}
	if !found {
		return result, errors.New("offline upgrade requires an existing operator public key; it does not alter access grants")
	}
	key, err := i.readInstalled(ctx, hostKeyPath, 16384)
	if err != nil || len(key) == 0 {
		return result, errors.New("existing SSH host identity is required; no rotation")
	}
	pub, err := i.command(ctx, "/usr/bin/ssh-keygen", "-y", "-f", hostKeyPath)
	if err != nil {
		return result, err
	}
	if _, err := access.AuthorizedKey(pub, "operator"); err != nil {
		return result, err
	}
	result = HostResult{Owner: p.Name, UID: p.UID, PierRoot: p.Piers, OperatorFingerprint: grant.Fingerprint, HostPublicKey: strings.Join(strings.Fields(string(pub))[:2], " "), SSHPort: SSHPort}
	lockPath := "/etc/urbit-runner/.setup.lock"
	if _, err := i.readInstalled(ctx, lockPath, 0); errors.Is(err, os.ErrNotExist) {
		if err := i.installFile(ctx, lockPath, nil, 0600, false); err != nil {
			return result, err
		}
	} else if err != nil {
		return result, err
	}
	setup, err := os.OpenFile(i.path(lockPath), os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return result, err
	}
	defer setup.Close()
	if err := syscall.Flock(int(setup.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another host installation/update is running")
	}
	socket := fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", p.UID)
	target := executableGeneration{SHA256: digestBytes(binary), MinSchema: 11, Schema: managed.StateSchemaVersion}
	var j *activationJournal
	data, err := i.readInstalled(ctx, activationJournalPath, 16384)
	if err == nil {
		var previous activationJournal
		if json.Unmarshal(data, &previous) != nil || previous.Version != 1 || previous.Owner != p.Name || previous.UID != p.UID || !validDigest(previous.Token) || !previous.Target.valid() {
			return result, errors.New("invalid activation journal; preserve it for recovery")
		}
		if previous.Stage != "complete" && (!previous.Offline || previous.Target != target) {
			return result, errors.New("unfinished activation must resume with its original update mode and retained installer generation")
		}
		if previous.Stage == "complete" && previous.Target != target {
			return result, errors.New("one-time offline bridge is complete; use the normal online install-host update path")
		}
		j = &previous
		if j.Offline && (j.Previous == nil || j.Previous.Schema != 19 || j.RunnerID == "") {
			return result, errors.New("offline activation journal lacks historical generation/runner provenance")
		}
		if _, err := i.retainedBytes(ctx, j.Target); err != nil {
			return result, err
		}
		if j.Previous != nil {
			if _, err := i.retainedBytes(ctx, *j.Previous); err != nil {
				return result, err
			}
		}
		if j.Stage == "starting" || j.Stage == "verified" || j.Stage == "complete" {
			// A new invocation may already own the listener after a lost start or
			// release acknowledgement. Verify it before any stopped-state path.
			if _, proofErr := i.serviceEvidence(ctx, p.Name); proofErr == nil {
				status, err := i.finishActivation(ctx, socket, j, j.Target, "complete")
				if err != nil {
					return result, err
				}
				if err := i.installPackageHooks(ctx, binary); err != nil {
					return result, err
				}
				result.RunnerID, result.Mode, result.Ready = status.RunnerID, status.Mode, true
				return result, nil
			}
		}
		if !j.Offline {
			return result, errors.New("this completed installation uses online quiescence; start its manager and rerun install-host without --offline-upgrade")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err := i.managementStopped(ctx, p.Name, socket); err != nil {
		return result, fmt.Errorf("--offline-upgrade requires urbit-runner.service already stopped and no foreground listener: %w", err)
	}
	directory := filepath.Join(p.Home, ".local/share/urbit-runner/managed")
	acquire := managed.LockOfflineState
	if i.offlineLock != nil {
		acquire = i.offlineLock
	}
	lock, err := acquire(i.path(directory), p.UID)
	if err != nil {
		return result, err
	}
	defer func() {
		if lock != nil {
			_ = lock.Close()
		}
	}()
	if err := i.managementStopped(ctx, p.Name, socket); err != nil {
		return result, err
	}
	current, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
	if err != nil {
		return result, err
	}
	request := managed.OfflineUpdateRequest{Action: "inspect", Directory: directory}
	if j != nil && (j.Stage == "verified" || j.Stage == "complete") {
		if digestBytes(current) != j.Target.SHA256 {
			return result, errors.New("forward recovery requires the verified installed generation")
		}
		request.Action, request.RunnerID, request.Token, request.Schema = "inspect-continuation", j.RunnerID, j.Token, j.Target.Schema
	}
	state, err := i.offlineState(ctx, lock, p, request)
	if err != nil {
		return result, err
	}
	if j == nil {
		if state.Schema != 19 || state.Token != "" {
			return result, errors.New("initial offline upgrade requires unfenced schema19 state; current-schema recovery requires its retained journal")
		}
		schema, err := i.historicalSchema(ctx, current, p)
		if err != nil {
			return result, err
		}
		if schema != state.Schema {
			return result, errors.New("installed executable schema does not match the existing state schema")
		}
		previous := executableGeneration{SHA256: digestBytes(current), MinSchema: 19, Schema: 19}
		if err := i.retainGeneration(ctx, current, previous); err != nil {
			return result, err
		}
		if err := i.retainGeneration(ctx, binary, target); err != nil {
			return result, err
		}
		token := make([]byte, 32)
		if _, err := rand.Read(token); err != nil {
			return result, err
		}
		j = &activationJournal{Version: 1, Owner: p.Name, UID: p.UID, RunnerID: state.RunnerID, Token: hex.EncodeToString(token), Target: target, Previous: &previous, Offline: true}
		if err := i.saveActivation(ctx, j, "offline-prepared"); err != nil {
			return result, err
		}
	}
	if state.RunnerID != j.RunnerID || digestBytes(current) != j.Previous.SHA256 && digestBytes(current) != j.Target.SHA256 {
		return result, errors.New("offline state identity or installed artifact changed; no activation performed")
	}
	if progress != nil {
		progress("offline-fence")
	}
	switch j.Stage {
	case "offline-prepared", "offline-fenced", "starting":
		if state.Schema == 19 && j.Stage != "offline-prepared" {
			return result, errors.New("offline schema regressed; never lower schema markers during recovery")
		}
		request.Action, request.RunnerID, request.Token = "fence", j.RunnerID, j.Token
		i.mutated = true // commit may succeed even if its acknowledgement is lost
		state, err = i.offlineState(ctx, lock, p, request)
		if err != nil {
			return result, err
		}
		if state.Schema != target.Schema || state.RunnerID != j.RunnerID || !state.Quiesced || state.Token != j.Token {
			return result, errors.New("offline schema/fence commit was not verified")
		}
		if err := i.saveActivation(ctx, j, "offline-fenced"); err != nil {
			return result, err
		}
	case "verified", "complete":
		if state.Schema != target.Schema || state.Token != j.Token || digestBytes(current) != target.SHA256 || j.Stage == "complete" && state.Quiesced {
			return result, errors.New("verified offline activation continuation mismatch")
		}
	default:
		return result, errors.New("unknown offline activation stage")
	}
	if err := lock.Verify(); err != nil {
		return result, err
	}
	if err := i.managementStopped(ctx, p.Name, socket); err != nil {
		return result, err
	}
	if j.Stage != "verified" && j.Stage != "complete" {
		data, err := i.retainedBytes(ctx, j.Target)
		if err != nil {
			return result, err
		}
		if err := i.installFile(ctx, access.ShellPath, data, 0755, true); err != nil {
			return result, err
		}
		installed, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
		if err != nil || digestBytes(installed) != j.Target.SHA256 {
			return result, errors.New("activated executable verification failed; preserve fence and rerun the retained installer")
		}
		if err := i.saveActivation(ctx, j, "starting"); err != nil {
			return result, err
		}
	}
	// Only now may the runtime flock be released: every old binary is schema-
	// fenced, and the installed new binary honors the durable admission fence.
	if err := lock.Close(); err != nil {
		return result, err
	}
	lock = nil
	manager := func(args ...string) error {
		_, err := i.command(ctx, "/usr/bin/systemctl", append([]string{"--user", "--machine=" + p.Name + "@.host"}, args...)...)
		return err
	}
	if err := manager("daemon-reload"); err != nil {
		return result, err
	}
	start := []string{"enable", "--now", "urbit-runner.service"}
	if j.Stage == "verified" || j.Stage == "complete" {
		start = []string{"start", "urbit-runner.service"}
	}
	if err := manager(start...); err != nil {
		return result, fmt.Errorf("offline activation interrupted; schema19 rollback is forbidden; rerun this --offline-upgrade command: %w", err)
	}
	status, err := i.finishActivation(ctx, socket, j, j.Target, "complete")
	if err != nil {
		return result, fmt.Errorf("offline activation not verified; retain state/journal/generations and rerun --offline-upgrade: %w", err)
	}
	if err := i.installPackageHooks(ctx, binary); err != nil {
		return result, err
	}
	result.RunnerID, result.Mode, result.Ready = status.RunnerID, status.Mode, true
	return result, nil
}
