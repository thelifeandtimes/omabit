# Runtime qualification record

Status: rootless setup, isolated lifecycle, and host-loopback HTTP are accepted by
the user. This is still partial qualification, not production-runtime support.

## Accepted engine setup

The user reported successful checkpoint 2A installation and checks:

- Dedicated account `urbit-runner`, UID/GID **959/959** on this host only.
- Service `urbit-docker.service`; socket `/run/user/959/urbit-docker.sock`.
- Installer result: `account_created: true`, `rootless_verified: true`,
  `runtime_qualified: false`, `partial: false`.
- Subsequent read-only metadata inspection found Docker 29.7.2, RootlessKit 3.1.0,
  and slirp4netns 1.3.5 installed. No developer Docker API call was made.

This established the engine setup gate. Container/runtime evidence is recorded
separately below; Ames and a separately observed reboot remain outstanding.

## Accepted isolated lifecycle (checkpoint 2B)

User-supplied report for run `35aac1083d71188d9782ec64b49ba771`:

- Every phase passed: engine, image, pill, probe, boot, restart, pack, post-pack,
  and cleanup.
- `checkpoint_passed: true`, `cleanup_complete: true`, `limits_observed: true`,
  `pier_metadata_seen: true`; `runtime_qualified: false` as designed.
- All five containers were removed; every create outcome was resolved.
- Docker 29.7.2 reported the pinned manifest as its backend image ID; Vere
  reported `4.6-5a7614f`.
- Available memory at admission: 7,298,469,888 bytes; available disk:
  453,037,428,736 bytes. These are observations from that run, not reservations.
- The verified 217,421,172-byte pill matched the generation and SHA-256 below.
- The fake pier and report remain in the account's private qualification storage.
  No existing pier was adopted.

This is operator-reported execution evidence, not an independent developer rerun.

Checkpoint 1 has been accepted by the user. The qualification pill will initially
be the upstream default bootstrap pill for the selected Vere version, not the
unreleased Omabit pill. Record its resolved URL and content digest for repeatable
checks; verify the expected base, webterm, and groups desks after boot.

Before installation, read-only preflight found x86-64 Linux, systemd, and Docker
available. The rootful Docker socket was enabled but inaccessible to the desktop
user, as intended by Quattro. Rootless setup scripts, RootlessKit, and a networking backend
were absent. No engine was activated and no host configuration was changed by
that preflight.

The user confirmed dedicated-account rootless Docker as the default. Preserve
Quattro's existing system Docker services and the desktop's context and group
membership. Rootless installation has its own reviewed OS-authorization step;
there is no automatic privileged fallback.

Run only after the protocol walkthrough is accepted and the user approves any
host setup. Use disposable development ships. Never boot an existing identity's
copy while its original is running.

## Accepted host HTTP (checkpoint 2C)

User-supplied report for run `e9f766ae9854e04f35eb9991d39fb9ba`:

- Engine, image, pill, probe, network, host-http-boot, host-http-restart, and
  cleanup all passed.
- `checkpoint_passed: true`, `cleanup_complete: true`, `limits_observed: true`,
  and `pier_metadata_seen: true`; `runtime_qualified: false` as intended.
- Host endpoints were `http://127.0.0.1:32768/~/host` and then
  `http://127.0.0.1:32769/~/host`, both verified. Those containers are now removed;
  the addresses are historical evidence, not active servers.
- All three containers and network
  `fda5e86036dc00a5c9ed134ab3c41ac33f162174136bef5d6b84ee49bc8802b0` were removed;
  every create outcome was resolved.
- Admission observed 5,396,324,352 bytes available RAM and 452,211,556,352 bytes
  available disk. Image, Vere, and pill pins matched the accepted 2B candidate.

This confirms local publication/reconnect on the dedicated NAT bridge. It does
not establish remote/tailnet access, application authentication, or Ames.

## Record before testing

- Host distribution, architecture, systemd, kernel, Docker version.
- Dedicated account and rootless engine setup, including unattended startup.
- Image reference/digest, embedded Vere version, entrypoint and supported flags.
- Pill release/version, trusted metadata source, URL, digest, included desks.
- Pier/artifact paths, ownership, limits, port bindings, and network policy.
- Control-socket bug/fix status for the exact selected runtime build.
- Commands, results, and redacted logs for every check below.

## Checklist

| Area | Required observation | Result |
| --- | --- | --- |
| Isolation | No desktop Docker-group grant or rootful socket exposure | Accepted in checkpoint 2A (operator-reported) |
| Image | Correct x86-64 binary and explicit invocation | Passed in 2B |
| URL pill | HTTPS source downloads and verifies before boot | Passed in 2B |
| Local pill | Explicit import verifies before boot | Not run |
| Download errors | Missing source, redirect policy, oversize, timeout, and bad hash fail closed | Not run |
| Bootstrap | Expected identity mode and desks; no unrelated personal state | Fake boot passed; desk/content checks pending |
| Persistence | Stop/start preserves pier; containers are replaceable | Local replacement restart passed in 2B; application data semantics pending |
| Networking | Private HTTP access and intended Ames connectivity | In-container HTTP passed in 2B; host-loopback publication/reconnect passed in 2C; Ames/remote access pending |
| Shutdown | Graceful signal handling and adequate stop timeout | Passed in 2B |
| Reboot | Engine/runner start without desktop login | Not run |
| Offline pack | Exclusive pier access and successful restart | Passed in 2B |
| Offline meld | RAM admission and successful restart | Not run |
| Metrics | Correct capacity/open/idle units and per-metric freshness | Not run |
| Resource limits | Live memory, swap, PID, and CPU cgroup values | Observed in 2B; overload/failure enforcement tests pending |
| Socket safety | In-flight disconnect and collector termination do not crash runtime | Not run |
| Failures | Low disk, OOM, killed job, and invalid pier are bounded and diagnosable | Not run |

## Preparation evidence

- New `preflight` and `setup-plan` commands inspected this host without subprocess,
  Docker API, or network use and identified the missing rootless prerequisites.
- `prepare-rootless` fetched the immutable Docker 29.7.2 launcher and legal files
  over HTTPS and verified their compiled-in content pins.
- Generated scripts passed `sh -n`; systemd static verification only complained
  that the intended installed executable paths do not yet exist.
- Fixture tests cover the elevated first-install path without executing actual
  administrative commands. Bundle contents are reverified independently of its
  manifest before any host mutation.
- No packages, accounts, services, Docker contexts, group membership, or piers
  have been changed during developer validation. The existing rootful service
  remains inactive/disabled and its socket remains active/enabled.

Candidate dependency baseline: package-managed Docker 29.7.2, repository
RootlessKit and slirp4netns, and the launcher from Moby commit
`6a43e3d5afddf4111da0f864bbc7cae5d7e95001`. A changed engine version requires a new
reviewed baseline, not a bypass or a system downgrade. Preserve the package
manager's normal security-update workflow.

## Gates

Checkpoint 2C introduces a separate host-HTTP profile. It uses a dedicated normal
NAT bridge (with outbound access), not `--internal`, because the pinned Docker
gateway path is needed for NAT port publication. Only `127.0.0.1` and an ephemeral
TCP host port may be published. HTTP identity is checked directly from the host,
without credentials or redirects. Container replacement rediscovers the binding;
cleanup tracks both container and network intents. The accepted live execution
is recorded above.

Checkpoint 2B has fixed image/pill provenance, injected tests, and the accepted
execution record above. The image is pinned to
`sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd`;
the default 4.6 pill is pinned to SHA-256
`f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b` and
217,421,172 bytes, using object generation `1790171686269740`. The mutable default
was replaced during development; the old object was no longer publicly
retrievable. The new generation's hash was independently computed from its full
HTTPS content; no separate publisher signature was verified.

Do not enable continuous socket collection on an affected runtime. A draining
client is mitigation, not proof against collector hard-kill/OOM.

If rootless Docker does not meet a required capability, capture the evidence and
ask before installing a privileged alternative. Do not change existing Docker
membership, start real ships, or configure system services as an implicit test.

Epoch pruning needs its own later format/recovery qualification. Ordinary startup
success is not enough to claim retained history remains replayable.
