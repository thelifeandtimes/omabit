# Distribution-native host compatibility

This describes **RC.13**, including schema21. Published RC.12 binaries retain their
original Docker requirement. Building this source does not install packages or
update an existing host. See the [RC.13 notes](release-rc13.md) for qualification scope.

## Responsibilities

- **Runner:** owns local evidence, compatibility policy, package authentication
  and operation admission. Headless and desktop hosting share these checks.
- **Collector:** retrieves identity-bound reports and checks RPC capabilities.
  A collector-only machine needs no local Docker stack.
- **Omabit:** presents local/remote reports and native setup guidance in existing
  setup/monitoring areas. It does not impose its local distro on remote hosts.
- **Native providers:** Pacman and Debian/Ubuntu adapters supply metadata and
  authentication evidence. Unknown providers remain explicitly unresolved.

`internal/hostcompat` is independent of Omarchy and the runner/collector packages.
Host setup, runtime admission and the generated startup guard share its policy.
The pinned rootless launcher's artifact identity is independent of engine support.

## Compatibility, provenance and qualification

New-work policy accepts stable Docker **27.2.0 through 29.x**, including SemVer
build metadata such as `+dfsg1`, subject to actual API1.47 support, private-socket
identity, rootless operation, systemd cgroup v2 and the expected data root.
Prereleases, malformed versions and unsupported families do not authorize new
runtime work. Hosting remains Linux/AMD64 with systemd/cgroup v2.

Docker29.9.1 is **contract-compatible, not automatically qualified**. Its advertised
API range must include1.47. Vere content pins, ownership, mounts, commands, network
contracts and resource limits remain checked.

Reports distinguish installed full package versions, running daemon version,
provenance, compatibility and exact-stack qualification. Compatible package/daemon
drift is advisory `restart_pending`. Separate `can_observe`, `can_stop` and
`can_mutate` decisions preserve identity-verified control when package evidence
needs attention. Hard API/ownership/rootless/data-root failures still refuse
affected work. Host failures do not manufacture pier damage or disable the journal;
real existing provisioning/maintenance uncertainty remains authoritative.

Historical provisioning versions are not rewritten. New container intents record
the actual engine version. Qualification requires trusted exact stack/profile,
package and API evidence. The default remains unqualified without such a matching
record; compatible is not a claim of live testing of every combination.

## Local and fleet reports

These commands require RC.13 or later binaries:

```sh
# Metadata only: no engine connection, download or host changes.
urbit-runner preflight
urbit-runner doctor
urbit-runner host-check --format text

# As the installed pier owner: query only that owner's private engine.
urbit-runner host-check --live --format json

# Any collector machine; no local hosting prerequisites.
urbit-collector --runner server --format text host-check
urbit-collector --runner server host-check --format json
```

Authenticated read-only `GET /v1/host-compatibility` is advertised as
`host-compatibility-v1`. The envelope binds runner identity/mode and report version,
observation and expiry. Observer/operator grants can read it; mutation methods and
parameters are refused. Private data-root paths and raw tool diagnostics are not
returned remotely. Root verification/repair has no remote management endpoint.
Successful command exit means a report was generated; readiness must be read from
the report rather than inferred from exit0 or a version string.

Native evidence has at most 30-second cache freshness; live engine evidence at
most five seconds. Refreshes use bounded deadlines and single-flight workers.
Report failure preserves usable ship snapshots. Older runners remain observable
with feature-upgrade guidance. Metadata-only or stale reports are diagnostic,
not live action authority. Fleet polling and Omabit Refresh revalidate normally.

## Native authentication

Pacman uses trusted native GnuPG validation of detached or repository-carried
`PGPSIG` signatures, exact package identity, and installed executable SHA-256
matching authenticated payloads. Recorded package-manager validation is labelled
as an assertion, not substituted for cryptographic verification.

Debian/Ubuntu verifies source-scoped `Signed-By` authority → InRelease → Packages
SHA-256 → exact package/version/architecture and `.deb` SHA-256 → installed
executable bytes. Supported `.list` and Deb822 `.sources` include file-based `.gpg`
and armored `.asc` keys. Native source/config selection and removal affect
authorization. Unsupported authority overrides and insecure bypasses fail closed.
Individual `.deb` files and dpkg MD5 records are not treated as signatures.

```sh
sudo urbit-runner verify-host-packages --acknowledge-local-verification
# Explicitly retrieve missing artifacts for the exact installed native builds:
sudo urbit-runner verify-host-packages --acknowledge-local-verification --fetch-missing
```

Default verification is offline. Fetching uses bounded private staging and native
sources, without package installation/update, repository changes or key imports.
APT acquisition runs as its existing non-root `_apt` user; completed downloads
are sealed before authentication. Missing historical builds or native tools are
reported, never replaced with another version. Global proxy/XferCommand customizations
are excluded from this private fetch path. Limits are two minutes offline and ten
minutes with fetching, subject to any shorter caller deadline.

Root-issued public receipts are at
`/var/lib/omabit/hostcompat/receipts-v2.json`; retained evidence is private under
`evidence-v2/`. Non-root readers need neither private GnuPG databases nor transient
OS archives. Cache cleanup and APT-list rotation do not invalidate unchanged
authenticated payloads; idle time alone does not expire authorization. Changes to
payloads, package identity or public authority evidence invalidate affected proof.
Manual private Pacman trust edits require root refresh. Hash caching is bound to
filesystem identity and change metadata, with aggregate I/O limits.

## OS update integration

Successful authorized hosting setup/management activation installs:

- `/etc/pacman.d/hooks/00-omabit-host-packages.hook`
- `/etc/apt/apt.conf.d/00-omabit-host-packages`
- `/usr/local/lib/urbit-runner/package-hooks/refresh-host-packages`

The native hooks verify and retain evidence before cache cleanup. Failure does
not abort an OS upgrade, restart services or touch piers; wrapper success is never
authentication evidence. Effective ordering/cache locations are checked, and
administrator edits are preserved. Custom roots, earlier cleanup or nondefault
cache layouts require reviewed support. See [hook details](../internal/hostinstall/PACKAGE-HOOKS.md).

## Existing hosts

Authorized updates migrate only exact known generated guards/units, retaining
the launcher/data paths, owner, SSH and state. An established stopped engine is
not started as a side effect. Interrupted initial setup has separate activation
evidence so it can resume. Existing valid subordinate-ID grants are preserved
without allocating more authority; fresh allocation retains the strict NSS policy.

If the old exact-version guard has already blocked the engine:

```sh
sudo urbit-runner repair-engine-guard --acknowledge-host-changes --fetch-missing
```

This verifies packages, changes only recognized guards/units, and reloads an active
user manager. It never starts/restarts Docker or clears update fences. Coordinate
any engine restart separately. Schema21 fences older runners; see the
[upgrade procedure and limits](host-compatibility-upgrades.md).

## Verification and support scope

Regression coverage includes actual Unix probes, RPC identity/roles/freshness,
collector-only use, version drift, guard migration, receipt tampering and atomic
publication, authority removal, signatures, interrupted setup/boot/update, historical
records and schema20→21 preservation. Offscreen QML consumes Go-serialized reports.

Native integration additionally exercised real APT/dpkg/GnuPG and Pacman transactions
inside disposable Ubuntu/Arch environments, using signed synthetic payload packages:
fetch, update hooks, cache cleanup and non-root receipt reads. These tests verify
package authentication, not live Vere on every new engine/distro. No custom engine
or package builds are distributed. RPM-family providers and further full-stack
runtime qualification remain separate support work.

The reusable [native qualification harness](../scripts/qualification/host-packages/README.md)
is retained with disposable-guest guards and unprivileged negative tests. The
[development verification record](diagnostics/host-compatibility-development.md)
identifies the executed tests and native evidence, including their scope limits.
