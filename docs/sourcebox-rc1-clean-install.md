# Fresh rc.1 runner installation on sourcebox

**Historical procedure only.** The current candidate walkthrough is
[sourcebox-clean-install.md](sourcebox-clean-install.md). Do not execute these old
rc.1 download pins or replay an already completed retirement.

**Version note:** this walkthrough pins the published rc.1 binaries. The newer
storage-location/import/delete GUI work is deployed as `0.1.0-rc.2-dev` on the desktop
and both runners (now schema 15, including unclean-reboot resume). To qualify those features during a clean install,
wait for their release package and update the pinned downloads/checksums in this
guide first; changing a URL alone does not verify a new release.
The published rc.1 runner also predates the merged btrfs identity fix in PR #1.
Sourcebox's current schema-15 build includes that fix. The older download refuses
the current database and, after a full reset, would install the older behavior.

This walkthrough retires the existing **sourcebox Omabit installation and all of
its managed ships**, then installs the published `0.1.0-rc.1` runner into an empty
account/home/engine. It creates a new runner ID, SSH host key and operator grant.
Use the existing sourceport desktop to enroll and test it.

The old files are moved into a root-only retirement directory, rather than reused
or immediately erased. This preserves the recent failure evidence while still
exercising first-time installation. It is not a supported backup/restore workflow.

**Scope:** only the dedicated `urbit-runner` installation on **sourcebox**. The local
runner/plugin, sourcebox's native/tmux ships, Tend, system Docker and shared package
dependencies stay in place. In particular, do not run a global Docker prune,
`tailscale serve reset`, `userdel -r`, or a broad deletion of Urbit processes/data.

This is a reviewed manual procedure, **not a previously live-qualified uninstaller**.
Run sections in order, stop on an error, and retain the output. If a section stops
partway through, inspect that state before continuing; do not rerun teardown from
the top. Plan for Omabit downtime on sourcebox and time to pull the image/bootstrap.
No host reboot is needed until the optional final reboot check.

## 1. Prepare the desktop client and a new operator key — sourceport

Use a Bash terminal on **sourceport**. Keep it open for sections 5–7. These variables
and directories are separate from the existing local setup and remote-test key.
If either new directory already exists, choose a new suffix consistently instead
of overwriting it.

```bash
bash
```

```bash
set -euo pipefail
umask 077
WORK="$HOME/.local/share/omabit-sourcebox-rc1"
ACCESS="$HOME/.config/omabit/sourcebox-rc1"
COLLECTOR="$WORK/urbit-collector"
RELEASE=https://github.com/thelifeandtimes/omabit/releases/download/v0.1.0-rc.1

test ! -e "$WORK"
test ! -e "$ACCESS"
mkdir -p "$HOME/.local/share" "$HOME/.config/omabit"
mkdir -m 0700 "$WORK" "$ACCESS"
curl --fail --location --output "$COLLECTOR" "$RELEASE/urbit-collector-linux-amd64"
printf '%s  %s\n' \
  b66ce1f742e7a82a59658be1487f472ff81a2976c39cc2dc129f76be2bdc1011 \
  "$COLLECTOR" | sha256sum --check -
chmod 0755 "$COLLECTOR"

ssh-keygen -q -t ed25519 -N '' -C omabit-sourcebox-rc1 -f "$ACCESS/id_ed25519"
ssh-keygen -lf "$ACCESS/id_ed25519.pub"
```

This unencrypted key is for unattended, restricted Omabit RPC, not administrator
SSH. Only its `.pub` file goes to sourcebox.

Capture the old fleet, check that no operation is active, and forget only the old
remote enrollment. The expected old runner ID below comes from the accepted
sourcebox installation. If it differs, inspect `runners` before proceeding.

```bash
"$COLLECTOR" runners
"$COLLECTOR" --runner remote-test \
  --expected-runner-id runner-b4043fa2d0aee501aa2cf78322aed911 \
  snapshot > "$WORK/remote-before.json"
python3 - "$WORK/remote-before.json" <<'PY'
import json, sys
s = json.load(open(sys.argv[1]))
assert s['runner_id'] == 'runner-b4043fa2d0aee501aa2cf78322aed911'
assert not any(i.get('active_operation_id') for i in s['instances']), 'Wait for active operations'
assert not any(o['status'] in ('queued', 'running') for o in s['operations']), 'Wait for queued/running operations'
for i in s['instances']:
    print(i['id'], i['observed_state'], 'recovery-required' if i.get('recovery_required') else '')
PY
"$COLLECTOR" forget remote-test \
  --runner-id runner-b4043fa2d0aee501aa2cf78322aed911
```

At guide preparation, the old fleet was `panel-check`, `rc1-pill-check`,
`remote-smoke` and `remote-comet`. The latter two were already recovery-required;
discarding this fleet does not require recovering them first. Refresh the desktop
fleet and finish any other operator sessions before teardown. The local moved-source
record for `panel-check` remains fenced; do not reactivate its archived source.

Prepare a new administrator-owned working directory remotely and copy the public key:

```bash
ssh sourcebox 'umask 077; mkdir -m 0700 "$HOME/omabit-clean-rc1"'
scp "$ACCESS/id_ed25519.pub" sourcebox:omabit-clean-rc1/operator.pub
```

Use the ordinary **administrator** `sourcebox` SSH alias, not the old restricted
`omabit-remote-test` alias. The administrator key may prompt for its passphrase.

## 2. Download and preflight the runner — sourcebox

Open a second terminal, connect using `ssh sourcebox`, then start Bash:

```bash
bash
```

All commands in sections 2–4 run in that **sourcebox administrator terminal**:

```bash
set -euo pipefail
umask 077
test "$(hostname -s)" = sourcebox
RC_DIR="$HOME/omabit-clean-rc1"
RELEASE=https://github.com/thelifeandtimes/omabit/releases/download/v0.1.0-rc.1
test -f "$RC_DIR/operator.pub"
cd "$RC_DIR"

curl --fail --location --output urbit-runner-linux-amd64 \
  "$RELEASE/urbit-runner-linux-amd64"
printf '%s  %s\n' \
  1467ef2c0edf73ccce81d6db798237e992006bebb9b58243768d0c5a43fe7e70 \
  urbit-runner-linux-amd64 | sha256sum --check -
chmod 0755 urbit-runner-linux-amd64
./urbit-runner-linux-amd64 preflight | tee preflight-before.json
./urbit-runner-linux-amd64 prepare-rootless --output "$RC_DIR/rootless-bundle"
free -h
df -h /var/lib/urbit-runner
```

Preflight exit zero means the report was produced; **read its checks**. Hosting
requires Linux AMD64, systemd/cgroup v2 and the supported Docker **29.7.2**,
RootlessKit, slirp4netns, UID-map helpers, iptables and OpenSSH. Private runtime
metadata may be unreadable to the administrator's ordinary user. Resolve platform
or package failures before retirement. Keep the existing shared packages installed.
Each ship start requires at least 5 GiB available RAM and 8 GiB free disk.

### Retire the old Omabit tailnet routes

```bash
sudo tailscale serve status --json > "$RC_DIR/tailscale-before.json"
sudo tailscale serve status
```

Review the routes. The recorded Omabit routes were TCP **22222**, **18081** and
**18082**, each forwarding to the same port on `127.0.0.1`. For those exact routes
that are still present and still belong to Omabit, turn them off:

```bash
sudo tailscale serve --bg --tcp=22222 off
sudo tailscale serve --bg --tcp=18081 off
sudo tailscale serve --bg --tcp=18082 off
```

Skip an already absent route; investigate a route whose target has changed. Leave
all other Serve entries in place. In particular, port 18080 was previously occupied
by an unrelated workload and is not part of this procedure.

## 3. Retire the old dedicated installation — sourcebox

This block stops only the dedicated account's services/processes, moves its files
out of the installer paths, and removes the account/group. It retains the old home,
Docker state, piers, SSH keys, configuration and journal excerpts below
`/var/lib/omabit-retired/sourcebox.XXXXXXXX`, accessible only to root.

Read the block before running it. The home must be a normal directory on the same
filesystem as the retirement directory; a separate mounted home needs an adapted
procedure. The script stops on leftover processes, mounts or subordinate-ID entries.

```bash
sudo /usr/bin/bash <<'SH'
set -Eeuo pipefail
export PATH=/usr/bin:/bin
umask 077
[[ $(hostname -s) == sourcebox && $EUID == 0 ]]
entry=$(getent passwd urbit-runner)
IFS=: read -r name password uid gid gecos home shell <<< "$entry"
[[ $name == urbit-runner && $home == /var/lib/urbit-runner ]]
[[ $uid =~ ^[0-9]+$ && $uid -ne 0 && $gid =~ ^[0-9]+$ && $gid -ne 0 ]]
[[ $(id -gn urbit-runner) == urbit-runner && $(id -G urbit-runner) == "$gid" ]]
[[ -d "$home" && ! -L "$home" ]]
[[ ! -L /var/lib/omabit-retired ]]
install -d -o root -g root -m 0700 /var/lib/omabit-retired
[[ $(stat -c %d "$home") == $(stat -c %d /var/lib/omabit-retired) ]]
retired=$(mktemp -d /var/lib/omabit-retired/sourcebox.XXXXXXXX)
printf 'Retirement directory: %s\n' "$retired"
trap 'printf "Stopped at line %s; inspect %s before continuing.\n" "$LINENO" "$retired" >&2' ERR

printf '%s\n' "$entry" > "$retired/account.txt"
getent group urbit-runner > "$retired/group.txt"
for db in subuid subgid; do
  awk -F: -v name=urbit-runner -v uid="$uid" '$1 == name || $1 == uid' \
    "/etc/$db" > "$retired/$db.txt"
done
journalctl _UID="$uid" _SYSTEMD_USER_UNIT=urbit-runner.service \
  -n 300 --no-pager > "$retired/runner-journal.txt"
journalctl _UID="$uid" _SYSTEMD_USER_UNIT=urbit-docker.service \
  -n 300 --no-pager > "$retired/docker-journal.txt"

# Stop entry points first; then let the dedicated engine shut down its containers.
systemctl disable --now urbit-runner-ssh.service
systemctl --user --machine=urbit-runner@.host disable --now urbit-runner.service
systemctl --user --machine=urbit-runner@.host disable --now urbit-docker.service
loginctl disable-linger urbit-runner
if loginctl show-user urbit-runner >/dev/null 2>&1; then
  loginctl terminate-user urbit-runner
fi
systemctl stop "user@${uid}.service" "user-${uid}.slice" "user-runtime-dir@${uid}.service"
if pgrep -u "$uid" >/dev/null; then
  printf 'Dedicated-account processes remain; inspect before moving files.\n' >&2
  exit 1
fi
mounts=$(findmnt -rn -o TARGET)
if awk '$0 == "/var/lib/urbit-runner" || index($0, "/var/lib/urbit-runner/") == 1 { found=1 } END { exit !found }' <<< "$mounts"; then
  printf 'Mounts remain at/below the old home; inspect before moving files.\n' >&2
  exit 1
fi

retire() {
  local path=$1
  if [[ -e "$path" || -L "$path" ]]; then
    mkdir -p "$retired$(dirname "$path")"
    mv -T -- "$path" "$retired$path"
  fi
}
retire /var/lib/urbit-runner
retire /etc/urbit-runner
retire /usr/local/lib/urbit-runner
retire /etc/systemd/user/urbit-runner.service
retire /etc/systemd/user/urbit-runner.service.d
retire /etc/systemd/user/urbit-docker.service
retire /etc/systemd/user/urbit-docker.service.d
retire /etc/systemd/system/urbit-runner-ssh.service
retire /etc/systemd/system/urbit-runner-ssh.service.d
retire "/etc/systemd/system/user@${uid}.service.d/omabit-shutdown.conf"
retire /run/urbit-runner-sshd.pid

# No -r or --force: the home is already retired and live-account refusal is useful.
userdel urbit-runner
if getent group urbit-runner >/dev/null; then groupdel urbit-runner; fi
for db in subuid subgid; do
  remaining=$(awk -F: -v name=urbit-runner -v uid="$uid" '$1 == name || $1 == uid' "/etc/$db")
  if [[ -n $remaining ]]; then
    printf 'Leftover dedicated mappings in /etc/%s; review before reinstalling.\n' "$db" >&2
    exit 1
  fi
done
systemctl daemon-reload
if getent passwd urbit-runner >/dev/null || getent group urbit-runner >/dev/null; then
  printf 'The old account/group still exists.\n' >&2
  exit 1
fi
for path in /var/lib/urbit-runner /etc/urbit-runner /usr/local/lib/urbit-runner \
  /var/lib/systemd/linger/urbit-runner \
  /etc/systemd/user/urbit-runner.service /etc/systemd/user/urbit-docker.service \
  /usr/lib/systemd/user/urbit-docker.service /usr/lib/systemd/user/urbit-docker.service.d \
  /run/systemd/user/urbit-docker.service /run/systemd/user/urbit-docker.service.d; do
  [[ ! -e "$path" && ! -L "$path" ]]
done
printf 'Old Omabit installation retired to %s\nFresh account installation can now proceed.\n' "$retired"
SH
```

Docker shutdown can take several minutes within its configured grace period. If
the block fails, keep the printed retirement path and error; resolve the specific
remaining resource rather than forcing account removal or deleting shared files.
The retired files are for inspection and eventual disposal, not for copying into
the fresh home. In particular, do not restore the old SQLite database or moved piers.

## 4. Install the fresh headless host — sourcebox

In the same administrator Bash terminal:

```bash
cd "$RC_DIR"
./urbit-runner-linux-amd64 preflight | tee preflight-clean.json
sudo ./urbit-runner-linux-amd64 install-host \
  --bundle "$RC_DIR/rootless-bundle" \
  --public-key "$RC_DIR/operator.pub" \
  --acknowledge-host-changes | tee "$RC_DIR/install.jsonl"
```

Before installation, the preflight should report the dedicated account as missing.
Installation should create it and finish with a `host.ready: true` receipt. Extract
that **public** receipt, verify that the runner identity is new, and check services:

```bash
python3 - "$RC_DIR/install.jsonl" "$RC_DIR/host.json" <<'PY'
import json, pathlib, sys
rows = [json.loads(line) for line in pathlib.Path(sys.argv[1]).read_text().splitlines() if line.strip()]
hosts = [r['host'] for r in rows if r.get('host', {}).get('ready') is True]
assert len(hosts) == 1, 'No unambiguous ready receipt; inspect install output'
h = hosts[0]
assert h['runner_id'].startswith('runner-')
assert h['runner_id'] != 'runner-b4043fa2d0aee501aa2cf78322aed911', 'Old runner state was reused'
assert h['mode'] == 'development-managed' and h['ssh_port'] == 22222
pathlib.Path(sys.argv[2]).write_text(json.dumps(h, indent=2) + '\n')
print(json.dumps(h, indent=2))
PY
sudo systemctl --user --machine=urbit-runner@.host is-active urbit-docker.service urbit-runner.service
sudo systemctl --user --machine=urbit-runner@.host is-enabled urbit-docker.service urbit-runner.service
sudo systemctl is-active urbit-runner-ssh.service
sudo systemctl is-enabled urbit-runner-ssh.service
loginctl show-user urbit-runner -p UID -p Linger
printf '%s  %s\n' \
  1467ef2c0edf73ccce81d6db798237e992006bebb9b58243768d0c5a43fe7e70 \
  /usr/local/lib/urbit-runner/current/urbit-runner | sha256sum --check -

sudo tailscale serve --bg --tcp=22222 tcp://127.0.0.1:22222
sudo tailscale serve status
```

Expect both user services and dedicated SSH active/enabled, plus `Linger=yes`.
The new account's numeric UID may or may not equal the old UID 954; use the actual
new value for diagnostics. This is a tailnet-only management route.

## 5. Pin and enroll the new identity — sourceport

Return to the sourceport Bash terminal from section 1. Copy the installer receipt
over the trusted **administrator** connection. Do not obtain the new management
host key using an unverified `ssh-keyscan` or by disabling host-key checking.

```bash
scp sourcebox:omabit-clean-rc1/host.json "$WORK/host.json"
python3 - "$WORK/host.json" "$ACCESS" <<'PY'
import json, pathlib, subprocess, sys
h = json.loads(pathlib.Path(sys.argv[1]).read_text())
access = pathlib.Path(sys.argv[2])
assert h['ready'] and h['runner_id'] != 'runner-b4043fa2d0aee501aa2cf78322aed911'
fingerprint = subprocess.check_output(['ssh-keygen', '-lf', str(access / 'id_ed25519.pub')], text=True).split()[1]
assert fingerprint == h['operator_fingerprint'], 'Installer did not grant the expected new key'
key = h['host_public_key'].split()
assert len(key) == 2 and key[0] == 'ssh-ed25519' and h['ssh_port'] == 22222
with (access / 'known_hosts').open('x') as f:
    f.write('[sourcebox.tail709ccc.ts.net]:22222 ' + ' '.join(key) + '\n')
with (access / 'ssh_config').open('x') as f:
    f.write(f'''Host omabit-sourcebox-rc1
  HostName sourcebox.tail709ccc.ts.net
  Port 22222
  User urbit-runner
  IdentityFile "{access / 'id_ed25519'}"
  IdentityAgent none
  IdentitiesOnly yes
  UserKnownHostsFile "{access / 'known_hosts'}"
  GlobalKnownHostsFile /dev/null
  StrictHostKeyChecking yes
  UpdateHostKeys no
  CheckHostIP no
  BatchMode yes
  ForwardAgent no
''')
print('Pinned the new management host key and verified the operator fingerprint.')
PY
chmod 0600 "$ACCESS/known_hosts" "$ACCESS/ssh_config"
RUNNER_ID=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["runner_id"])' "$WORK/host.json")
CONNECTION=sourcebox-rc1
"$COLLECTOR" enroll "$CONNECTION" --ssh omabit-sourcebox-rc1 --ssh-config "$ACCESS/ssh_config"
"$COLLECTOR" --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" hello
"$COLLECTOR" --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" snapshot > "$WORK/empty.json"
python3 - "$WORK/empty.json" "$RUNNER_ID" <<'PY'
import json, sys
s = json.load(open(sys.argv[1]))
assert s['runner_id'] == sys.argv[2]
assert not s['instances'] and not s['operations'] and not s['incidents'], 'Runner is not empty'
assert not s.get('piers', {}).get('candidates', []), 'Unexpected old pier data'
print('PASS: new pinned runner, empty fleet, empty history and no discovered old piers.')
PY
```

Hello should show `access_role: operator`, `mode: development-managed`, and feature
`bootstrap-selection`. Refresh Omabit; `sourcebox-rc1` should appear alongside the
existing local connection. A changed-identity error is a stop-and-inspect result,
not a reason to weaken SSH checking or reuse the old runner ID.

## 6. Exercise the clean installation — sourceport and sourcebox

Start with **bare bootstrap**, so the known draft-app issues do not obscure the
basic installation check. The explicit rc.1 collector below also works while the
installed desktop is still development 0.22.3.

In the **sourceport** terminal, define a convenience function:

```bash
remote() { "$COLLECTOR" --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" "$@"; }
remote create-comet fresh-bare --http-port 18081 \
  --request-id clean-rc1-bare-register --confirm-network --bootstrap bare
remote operate fresh-bare start --revision 1 --request-id clean-rc1-bare-start
"$COLLECTOR" --format text --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" watch --interval 2s
```

Registration alone is stopped/uninitialized. Start is asynchronous: wait for its
operation to succeed and the instance to become running/healthy, then press Ctrl+C
to end observation. First boot also pulls the pinned image into the empty engine.
If a mutation's acknowledgment is lost, inspect the journal/request ID before any
retry. A failed or recovery-required state needs inspection, not repeated Starts.

Save the first healthy identity and verify a stop/resume cycle:

```bash
remote snapshot > "$WORK/bare-first-boot.json"
remote logs fresh-bare --revision 1 > "$WORK/bare-first-boot-logs.json"
remote operate fresh-bare stop --revision 1 --request-id clean-rc1-bare-stop
"$COLLECTOR" --format text --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" watch --interval 2s
```

Wait for Stop to succeed and show stopped, then Ctrl+C. Only then run:

```bash
remote operate fresh-bare start --revision 1 --request-id clean-rc1-bare-resume
"$COLLECTOR" --format text --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" watch --interval 2s
```

After running/healthy returns, Ctrl+C and compare:

```bash
remote snapshot > "$WORK/bare-resumed.json"
python3 - "$WORK/bare-first-boot.json" "$WORK/bare-resumed.json" <<'PY'
import json, sys
def ship(path):
    return next(i for i in json.load(open(path))['instances'] if i['id'] == 'fresh-bare')
a, b = map(ship, sys.argv[1:])
assert a['health'] == b['health'] == 'healthy'
assert a['ship'] == b['ship'] and a['pier_path'] == b['pier_path']
assert b['observed_state'] == 'running' and not b.get('recovery_required')
assert b['config_revision'] == b['applied_revision'] == 1
assert b['runtime_status']['container_kind'] == 'comet-resume'
assert b['runtime_status']['container_id'] != a['runtime_status']['container_id']
print('PASS: clean boot and initialized replacement retained', b['ship'])
PY
```

### Browser path

On **sourcebox**, expose only the new ship's chosen port:

```bash
sudo tailscale serve --bg --tcp=18081 tcp://127.0.0.1:18081
```

On **sourceport**:

```bash
curl --fail --silent --show-error http://sourcebox.tail709ccc.ts.net:18081/~/host
```

The returned identity must match `fresh-bare`. In **Manage runners**, set the new
connection's Browser host to `http://sourcebox.tail709ccc.ts.net` (no port; Omabit
preserves the ship's port). Use Open ship and Copy +code to test login. Keep codes
out of shell history and shared logs.

### Recommended Apps follow-up

After the bare lifecycle succeeds, test the released pill download with a **second**
new comet on sourceport:

```bash
PILL=https://github.com/thelifeandtimes/omabit/releases/download/v0.1.0-rc.1/omabit-recommended-apps-0.1.0-rc.1.pill
remote create-comet fresh-recommended --http-port 18082 \
  --request-id clean-rc1-apps-register --confirm-network \
  --bootstrap recommended --pill "$PILL"
remote snapshot > "$WORK/recommended-registered.json"
python3 - "$WORK/recommended-registered.json" <<'PY'
import json, sys
i = next(i for i in json.load(open(sys.argv[1]))['instances'] if i['id'] == 'fresh-recommended')
assert i['bootstrap'] == {'kind': 'recommended', 'pill': {
    'sha256': '51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e',
    'bytes': 57011409}}
assert i['observed_state'] == 'uninitialized'
print('PASS: release URL staged the expected frozen pill.')
PY
remote operate fresh-recommended start --revision 1 --request-id clean-rc1-apps-start
"$COLLECTOR" --format text --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" watch --interval 2s
```

Wait for success/health, Ctrl+C, capture logs and leave this second ship stopped:

```bash
remote logs fresh-recommended --revision 1 > "$WORK/recommended-boot-logs.json"
remote operate fresh-recommended stop --revision 1 --request-id clean-rc1-apps-stop
"$COLLECTOR" --format text --runner "$CONNECTION" --expected-runner-id "$RUNNER_ID" watch --interval 2s
```

Wait for successful Stop, then Ctrl+C. This draft lacks MCP/Omachan and previously
logged Grubbery compilation and Azimuth snapshot/HTTP failures. Record application
results separately from process/identity/HTTP health; see the
[rc.1 qualification](diagnostics/2026-10-10-rc1-pill.md).

## 7. Service restart and optional host reboot

On **sourceport**, save a baseline with `fresh-bare` healthy/running and
`fresh-recommended` intentionally stopped:

```bash
remote snapshot > "$WORK/before-service-restart.json"
```

On **sourcebox**, restart management only:

```bash
sudo systemctl --user --machine=urbit-runner@.host restart urbit-runner.service
```

On **sourceport**, collect the next snapshot:

```bash
remote snapshot > "$WORK/after-service-restart.json"
```

Confirm the same runner ID, ship identities, revisions and states, the **same
running container ID**, and no new startup-resume operation. Management replacement
should not cycle a running ship.

A full sourcebox reboot is a separate user-scheduled step because it affects your
native ships and other workloads too. When convenient, prepare those workloads as
you normally would and reboot manually. After reconnecting:

```bash
remote snapshot > "$WORK/after-reboot.json"
```

Expect `fresh-bare` to resume with the same identity and a new container through
one successful `startup-resume`; `fresh-recommended` should remain intentionally
stopped. Runner ID and management host key must stay unchanged. Nonzero-exit or
recovery-required instances should remain quarantined; record that result rather
than repeatedly restarting services. A reboot outcome is not implied by the earlier
management-only restart.

## Results to retain

- Sourcebox: `~/omabit-clean-rc1/{preflight-before.json,preflight-clean.json,install.jsonl,host.json}`.
- Sourceport: the snapshots and bounded logs under `~/.local/share/omabit-sourcebox-rc1/`.
- The root-only retirement directory printed during teardown.
- Whether bare bootstrap, initialized resume, browser login, Recommended Apps,
  management restart and the optional reboot each passed or failed.

The new operator private key stays under `~/.config/omabit/sourcebox-rc1/`; do not
include it or cookies/+codes in a shared report. Keep retired state root-only: it
contains the former SSH host key and ship data. After the fresh installation is
accepted, that exact retirement directory can be removed as a separate cleanup;
never copy its piers or old runner database into the new installation.

If a prerequisite, SSH pin, install step or runtime operation fails, retain its
receipt and the current state. The installer supports reviewed partial-install
reruns; deleting more state is not a generic retry procedure.
