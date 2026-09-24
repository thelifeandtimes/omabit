# Checkpoint 4B: verified SSH transport and key-attributed access

Accepted on the dedicated account, September 24 at 19:00 UTC. The user's Hello
reported operator `ssh-key:SHA256:eVDrU/23cFq0QD1D5uuM10niBgNueT+kVji4wXdJNYc`.
Alpha stayed running/healthy at revision 2. Two `register-ssh-check` requests
returned the same succeeded operation `op-77c182d25396669c51322c117e114b5a` with
that actor; the new instance remained uninitialized at revision/applied 1/0.
Readable inventory polling also passed. The foreground SSH listener was then
stopped by the user. The steps below remain the reproducible installation guide.

## Evidence and remaining gate

Automated validation ran the installed OpenSSH 10.5p1 client/server with ephemeral
Ed25519 keys, StrictModes, a loopback listener, the actual bridge binary, real
SQLite state, and an injected runtime. It verified two operator keys, an observer
key, actor-scoped request replay, cross-operator revision conflicts, client
disconnect/reconnect, SSH host-key mismatch, runner-ID mismatch, rejection of
arbitrary commands/forwarding, and revocation during an existing session. The
built collector also enrolled that SSH endpoint and selected it by name.

Reproduce from the checkout without sudo:

```bash
go build -o ./build/ ./cmd/...
OMABIT_TEST_OPENSSH=1 \
  OMABIT_RUNNER_BINARY="$PWD/build/urbit-runner" \
  go test -v ./internal/managed -run '^TestOpenSSHIntegration$' -count=1
```

The test writes temporary private fixtures below `build/`, cleans up keys and
processes, and reports a human-readable PASS summary. It does not change host SSH
configuration, account shells, or managed piers.

**Essential manual gate:** administrative installation and the dedicated account's
actual PAM/restricted-shell path. The agent's `sudo -n true` requires a password.
The system SSH service was inactive when inspected. This walkthrough starts a
separate **127.0.0.1:22222** SSH listener in the foreground. It qualifies desktop
collector access to the existing rootless runner. Remote/tailnet reachability
needs a separately selected interface and second host later.

## 1. Create a collector access key and review bundle

From the checkout in your ordinary desktop terminal:

```bash
ssh-keygen -t ed25519 -f "$HOME/.ssh/omabit-runner-4b"
ssh-add "$HOME/.ssh/omabit-runner-4b"

REVIEW="$PWD/build/ssh-4b-review"
./build/urbit-runner prepare-ssh \
  --output "$REVIEW" \
  --public-key "$HOME/.ssh/omabit-runner-4b.pub" \
  --role operator
```

Choose a passphrase when prompted. Use a new filename if it already exists.
If your terminal has no SSH agent, start one for that terminal with
`eval "$(ssh-agent -s)"`, then repeat `ssh-add`.

`prepare-ssh` prints a readable summary and creates only two inert review files:
`authorized_keys` and `sshd_config`. It refuses an existing output directory.
Read them with your editor or:

```bash
less "$REVIEW/authorized_keys" "$REVIEW/sshd_config"
```

The key has `restrict` and a forced command bound to its fingerprint and role.
The SSH account shell will be the root-owned runner executable, which only
accepts that forced command when the original SSH request is exactly
`urbit-runner-rpc-v1`. It provides no interactive shell or arbitrary execution.
The account keeps its existing password state. The standalone server permits
only public-key authentication and uses the existing PAM account/session checks.

## 2. Install the checkpoint binaries and public grant configuration

First ensure the live runner has no active operation:

```bash
collector4() {
  sudo runuser -u urbit-runner -- /usr/bin/env -i \
    HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
    /usr/local/lib/urbit-runner/checkpoints/4a/urbit-collector "$@"
}

collector4 --runner workstation snapshot |
  jq '.instances[] | {id, active_operation_id, observed_state}'
```

Press Ctrl+C in the **serve-fake management terminal** once no operation is active.
The ship continues independently. In the desktop terminal, from the checkout:

```bash
REVIEW="$PWD/build/ssh-4b-review"

sudo install -D -o root -g root -m 0755 ./build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/4b/urbit-runner
sudo install -D -o root -g root -m 0755 ./build/urbit-collector \
  /usr/local/lib/urbit-runner/checkpoints/4b/urbit-collector

sudo test ! -e /etc/urbit-runner/sshd_config && \
sudo test ! -e /etc/urbit-runner/authorized_keys && \
sudo test ! -e /etc/urbit-runner/ssh_host_ed25519_key && \
sudo install -d -o root -g root -m 0755 /etc/urbit-runner && \
sudo install -o root -g root -m 0644 "$REVIEW/authorized_keys" /etc/urbit-runner/authorized_keys && \
sudo install -o root -g root -m 0644 "$REVIEW/sshd_config" /etc/urbit-runner/sshd_config && \
sudo ssh-keygen -q -t ed25519 -N '' -f /etc/urbit-runner/ssh_host_ed25519_key
```

This is a first-install sequence. If an existence check or installation step
fails, inspect that state before repeating; do not overwrite an earlier setup.
The host key is a separate SSH server identity, not an Urbit key.

Validate the new standalone configuration, then set the restricted executable as
the service account's shell:

```bash
sudo /usr/bin/sshd -t -f /etc/urbit-runner/sshd_config && \
sudo usermod --shell /usr/local/lib/urbit-runner/checkpoints/4b/urbit-runner urbit-runner
```

In the management terminal, resume management with the new binary:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/4b/urbit-runner \
  serve-fake --acknowledge-development
```

In a **separate SSH-server terminal**, start the new foreground listener:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Keep both terminals open. If authentication fails, share the server's diagnostic
message; do not unlock the account, switch to a general-purpose shell, or change
PAM policy to force access.

## 3. Pin the host key and enroll as your desktop user

Back in the desktop terminal, from the checkout, set `REVIEW` again: shell
variables and helper functions are local to each terminal. Create the dedicated
client config inside the review directory. Its host key comes directly from the
administratively installed public key, rather than trusting an unauthenticated
network scan:

```bash
REVIEW="$PWD/build/ssh-4b-review"

HOST_KEY="$(sudo cat /etc/urbit-runner/ssh_host_ed25519_key.pub)" &&
  printf '[127.0.0.1]:22222 %s\n' "$HOST_KEY" > "$REVIEW/known_hosts"

cat > "$REVIEW/ssh_config" <<EOF
Host omabit-local-4b
  HostName 127.0.0.1
  Port 22222
  User urbit-runner
  IdentityFile $HOME/.ssh/omabit-runner-4b
  IdentitiesOnly yes
  UserKnownHostsFile $REVIEW/known_hosts
  GlobalKnownHostsFile /dev/null
EOF

collect() {
  /usr/local/lib/urbit-runner/checkpoints/4b/urbit-collector \
    --format text "$@"
}

collect enroll workstation --ssh omabit-local-4b --ssh-config "$REVIEW/ssh_config"
collect --runner workstation hello
collect inventory
```

These collector commands run as **your desktop user**, without `sudo` or `runuser`.
They use your desktop collector registry, separate from checkpoint 4A's temporary
service-account collector registry.

Expected readable output:

- Runner `runner-f496feed1f7b08f31a5d82b94ac2f5fd` on this host.
- Mode `development-managed-fake`, feature `ssh-key-grants`.
- Actor `ssh-key:SHA256:...`, access `operator`.
- Workstation available, alpha running/healthy, revision/applied `2 / 2`.

## 4. Verify attribution with a harmless registration

After checking the snapshot for existing names/ports:

```bash
collect --runner workstation create-fake ssh-check \
  --http-port 18084 --request-id register-ssh-check --label "SSH acceptance"
collect --runner workstation create-fake ssh-check \
  --http-port 18084 --request-id register-ssh-check --label "SSH acceptance"
collect --runner workstation snapshot
```

Expected: both requests return the same completed registration operation, with
the SSH-key actor. `ssh-check` remains uninitialized and consumes no ship runtime
memory. Use another name/port/request ID if already registered; preserve exact
values for the replay. Registration remains in the development journal (there is
still no delete/unregister API).

For continuous readable status:

```bash
collect inventory --watch --interval 2s
```

Ctrl+C stops the watcher. Please report the readable `hello`, `inventory`, and
registration/replay results. This is the next essential evidence needed before
using this host's SSH path for further live work.

## End of session and rollback

Stop the foreground SSH server with Ctrl+C when finished. The managed runner and
ship lifetimes follow the existing explicit-stop procedure. To disable SSH access
persistently, restore the original account shell:

```bash
sudo usermod --shell /usr/bin/nologin urbit-runner
```

The root-owned configuration and host key can remain for review. Removing a key's
line from `/etc/urbit-runner/authorized_keys` revokes it on the next request even
on an existing connection; already accepted jobs continue. Use an atomic file
replacement when updating grants. Full host administrators remain able to alter
grants and audit history; key attribution is not tamper-proof against them.
