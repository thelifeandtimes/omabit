# Collector and SSH validation, September 24

## Workstation evidence

The user's collector watch event at 04:41:37 UTC reported the enrolled
`workstation` available, runner
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`, alpha running/healthy at configuration
and applied revision 2, and no incidents. The runner observation was approximately
nine seconds old. Earlier operation history remained present.

The user authorized continued programmatic validation and asked to pause only for
essential intervention, with readable validation commands.

## Programmatic evidence

- `scripts/validate-collection.py`: two actual simulator processes, persistent
  collector profiles, distinct runner IDs, named mutation routing, independent
  collector revision conflicts, partial inventory after a runner stops, rejection
  of a replacement before mutation, and explicit re-enrollment recovery passed.
- OpenSSH client/server: **10.5p1**, OpenSSL **3.6.4**.
- Go: **1.27.0-X:nodwarf5**, linux/amd64.
- `TestOpenSSHIntegration`: an isolated unprivileged loopback sshd, ephemeral host
  and client keys, real SSH authentication/known_hosts verification, actual runner
  bridge and collector executables, real SQLite, and an injected runtime passed.
- Two operator keys were correctly attributed and had independent request-ID
  scopes. An observer could read but could not register an instance. Untrusted
  keys, stale configuration writes, wrong host keys, wrong runner IDs, arbitrary
  commands, and forwarding were rejected. Accepted work survived client exit;
  reconnect returned the original operation. Revocation blocked an already-open
  session while leaving another operator available.
- Bridge tests verified that client-supplied actor/authentication headers are
  discarded and that an older runner is refused **before** receiving a delegated
  mutation. Delegation is checked on the same local connection as the request.
- `go test ./...`, `go vet ./...`, and race tests for access, client, collection,
  managed runner, and both CLIs passed. Both development binaries were rebuilt.

The OpenSSH fixture initially exposed StrictModes refusing an authorized_keys
path below `/tmp`. It now uses a private temporary directory under the owned
`build/` tree, retaining StrictModes. Temporary keys and processes are cleaned up.
These results exercise real SSH transport and authentication, not real Docker
lifecycle through the dedicated account or remote/tailnet reachability.

## Essential intervention

`sudo -n true` returned `sudo: a password is required`. The system `sshd.service`
was inactive. Installing the new root-owned binaries/grants and replacing the
service account's nologin shell with the restricted RPC executable needs the
user's administrative authorization. The actual dedicated-account PAM path is
therefore the next live gate. No account shell, host SSH configuration, or managed
runtime was changed during the automated work.

Follow [checkpoint 4B](../checkpoints/07-ssh-access.md). Its standalone SSH server
binds only `127.0.0.1:22222`; collector commands offer `--format text` for readable
status and key-attribution output. After that gate, continue with generic
origin-bound Eyre sessions and subsequent application work.
