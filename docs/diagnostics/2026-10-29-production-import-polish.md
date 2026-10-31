# Production import and Ubuntu setup polishing — 2026-10-29

Status: tested source changes on `codex/production-import-polish`, based on
published0.3.2. No release artifact, installed plugin, service or ship was updated.
No management schema or wire-format change is required. The next release must
include matching desktop, collector and runner artifacts and normal upgrade gates.

## Findings and corrections

- **Rooftop:** Ubuntu22.04.3 LTS / x86_64, systemd and cgroup v2. Required runtime
  packages were already installed. Stock split-component APT entries were rejected
  as conflicting authority. Merging identical URI/suite/architecture/key scopes
  fixes that without broadening signing authority. A second failure was a valid
  70,841-byte `Provides` line in the signed Jammy universe index. Unselected fields
  now use the existing 1 MiB field budget; selected identity/hash/path fields keep
  their 4096-byte bound. Errors name unverified packages. The existing installer
  lists missing native packages for approval before installing them.
- **Large local imports:** the shared100,000,000,000-byte bound incorrectly limited
  local imports to the handoff protocol's limit. Local inventory/copy now permits
  8 TiB. Cross-host handoff still rejects larger contents before emitting its
  archive. The import deadline scales with reviewed logical bytes instead of
  expiring after a fixed30 minutes. Free-space, ownership, stopped-source, content
  verification, retained-source and recovery rules remain.
- **Import presentation:** progress uses the desktop accent/track and typography.
  After an exact successful receipt and fresh matching instance snapshot, the
  form clears and the monitor opens. Stale, failed, mismatched or reused-name
  receipts do not navigate. The imported ship remains stopped.
- **Application connections:** title, approval toggle and connect/disconnect action
  share one row. Long titles elide with a full hover label; recovery details can
  use a separate line.

## Evidence

The remote diagnostic used actual fixed hostcompat source with a temporary test
entry point. It read native APT indexes/signatures, cached package archives and
installed executable bytes as the ordinary user. It provided no store, fetch or
receipt callbacks. Native `gpgv`, `apt-helper` and `dpkg-deb` remained bounded. All
eight roles passed: Docker engine/CLI, containerd, runc, rootlesskit, slirp4netns,
and uidmap/newgidmap. The run completed in29.45 seconds without exhausting its
verification budget. Only its temporary executable was staged and removed.
No package installation, privileged receipt publication or completed onboarding
is claimed. No owner profile was visible at the canonical path.

Focused tests passed for hostcompat, pierstore, piercopy, hostinstall, remotesetup,
and import/recovery paths in runtimequalify/managed. Hostcompat and local
copy/store/remote-setup race checks passed; `go vet` passed for the modified Go
subsystems. Tests include signed component merging/tamper rejection, streamed
large unrelated package fields, parser bounds, exact package error reporting,
sparse local size boundaries and handoff refusal before output.

The actual synthetic512 GiB coordinator test preserved the source inode through
a same-filesystem rename, enrolled a stopped instance and made no Docker/Vere
calls. Full runtimequalify (68.844s) and managed (276.113s) suites passed. Real small cross-filesystem fixtures
exercise copy/verification and recovery. **A full >100 GB copy has not been run.**

Actual QML offscreen fixtures passed for import progress/completion, import
restoration and application connections. They use private HOME/XDG roots and
inert service adapters. Connection controls remain inline at400px content width.
JavaScript model and restoration tests plus native QML registry tests passed.
One restoration launch initially failed at the outer sandbox's namespace boundary;
the same private Bubblewrap fixture passed with the required launch permission.

## Operator implications

These changes are not yet in published0.3.2. Its local import limit remains100 GB.
Updating the matching runner is necessary for larger imports; updating QML alone
cannot change backend admission.

An unfinished remote attempt pins its original installer generation. Resume setup
will keep that helper. After a corrected release, a new attempt with a distinct
Advanced connection name can use the new helper and the same host/owner. Normal
installed-state checks and OS approval still apply. Preserve old attempt journals;
never delete management state to force a fresh installation.

The protected fleet list now explicitly includes `~lismyl-lismyl`, `~sarlev`,
`~sarlev-sarsen`, `~tocwex` and all their moons, alongside earlier protections.
Live imports remain manual. No tmux ships, `~/temp-ur` data or native DEV review
state was operated on. Combined measured DEV/evidence/lab storage is about30.4 GiB
of the40 GiB budget (known unreadable overlay work entries remain). This pass's
listed temporary UI evidence totals under8 MiB after removing the diagnostic
binaries; no VM,
full pier copy or repetitive qualification bundle was created.
