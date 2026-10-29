# Pin the gateway native addon to its Node runtime

The fresh VM verifies all eight package roles, installs its own runner/rootless
engine, and admits the explicit read-only gateway mounts. Gateway startup then
fails before its key directory or any peer exists. Clean VM shutdown takes370.74s.
Read-only inspection confirms no OOM, exit1, and unchanged base/overlay hashes.
The helper had retained Docker stdout only; startup errors went to stderr. Reading
that exact container's bounded JSON log, only after source-hash and no-key/no-peer
checks, identifies Ganache LevelDB's missing libc.musl-x86_64.so.1.

The pinned node-gyp-build source chooses musl when /etc/alpine-release exists,
even though this fixture supplies a complete glibc Node runtime. The staged gateway
now sets LIBC=glibc and PREBUILDS_ONLY=1 and verifies Node's actual glibc runtime,
selected addon path and SHA-256 before loading Ganache or generating keys.
A private namespace probe reproduces the default musl selection without loading
it, then loads the pinned glibc addon and Ganache and proves LevelDB close/reopen
persistence. Its Node/library hashes match the VM payload. No raw database value
is exported. An initial probe refused because a bind-mounted host sysfs described
the old network view; the corrected probe omits sysfs and inspects private proc
network data. Keep that refusal; a new network namespace alone does not rewrite
an existing sysfs mount's namespace view.

The applied guest-only Docker helper can explicitly merge stdout/stderr for the
fixed200-line logs request against one exact container. It keeps the4MiB bound,
private-only retention and generic error redaction.17 inert tests pass. The actual
gateway addon fix awaits a fresh Docker runtime test; no ship messaging pass is
claimed. M340's pending mount admission is now proven and M341's streaming verifier
passes fresh installation. The diagnosed zero-ship VM disk is retired after proof
retention. New immutable appliances will use a separate private lab directory;
evidence continues to contain code/public metadata only, and runtime disk admission
and reserve thresholds remain8GiB/2GiB. No production ship or plugin changed.
