# Runtime qualification record

Status: rootless engine setup accepted by the user; Vere qualification is next.
Do not interpret the checklist as supported runtime behavior.

## Accepted engine setup

The user reported successful checkpoint 2A installation and checks:

- Dedicated account `urbit-runner`, UID/GID **959/959** on this host only.
- Service `urbit-docker.service`; socket `/run/user/959/urbit-docker.sock`.
- Installer result: `account_created: true`, `rootless_verified: true`,
  `runtime_qualified: false`, `partial: false`.
- Subsequent read-only metadata inspection found Docker 29.7.2, RootlessKit 3.1.0,
  and slirp4netns 1.3.5 installed. No developer Docker API call was made.

This establishes the engine setup gate, not container resource enforcement,
Vere operation, Ames reachability, or a separately observed reboot.

Checkpoint 1 has been accepted by the user. The qualification pill will initially
be the upstream default bootstrap pill for the selected Vere version, not the
unreleased Omabit pill. Record its resolved URL and content digest for repeatable
checks; verify the expected base, webterm, and groups desks after boot.

Before installation, read-only preflight found x86-64 Linux, systemd, and Docker available. The rootful
Docker socket is enabled but inaccessible to the desktop user, as intended by
Quattro. Rootless setup scripts, RootlessKit, and a rootless networking backend
were absent. No engine was activated and no host configuration was changed by
that preflight.

The user confirmed dedicated-account rootless Docker as the default. Preserve
Quattro's existing system Docker services and the desktop's context and group
membership. Rootless installation has its own reviewed OS-authorization step;
there is no automatic privileged fallback.

Run only after the protocol walkthrough is accepted and the user approves any
host setup. Use disposable development ships. Never boot an existing identity's
copy while its original is running.

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
| Isolation | No desktop Docker-group grant or rootful socket exposure | Not run |
| Image | Correct x86-64 binary and explicit invocation | Not run |
| URL pill | HTTPS source downloads and verifies before boot | Not run |
| Local pill | Explicit import verifies before boot | Not run |
| Download errors | Missing source, redirect policy, oversize, timeout, and bad hash fail closed | Not run |
| Bootstrap | Expected identity mode and desks; no unrelated personal state | Not run |
| Persistence | Stop/start preserves pier; containers are replaceable | Not run |
| Networking | Private HTTP access and intended Ames connectivity | Not run |
| Shutdown | Graceful signal handling and adequate stop timeout | Not run |
| Reboot | Engine/runner start without desktop login | Not run |
| Offline pack | Exclusive pier access and successful restart | Not run |
| Offline meld | RAM admission and successful restart | Not run |
| Metrics | Correct capacity/open/idle units and per-metric freshness | Not run |
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

Do not enable continuous socket collection on an affected runtime. A draining
client is mitigation, not proof against collector hard-kill/OOM.

If rootless Docker does not meet a required capability, capture the evidence and
ask before installing a privileged alternative. Do not change existing Docker
membership, start real ships, or configure system services as an implicit test.

Epoch pruning needs its own later format/recovery qualification. Ordinary startup
success is not enough to claim retained history remains replayable.
