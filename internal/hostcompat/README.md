# Host compatibility core

The shared contract API remains intact. Additional integration APIs are:

```go
func VerifyCurrent(ctx context.Context) (Report, error)
func VerifyCurrentWithOptions(ctx context.Context, options VerificationOptions) (Report, error)
type VerificationOptions struct { FetchMissing bool }
func EngineVersionShell() string
func EngineVersionsEquivalent(a, b string) bool

const VerificationDirectory = "/var/lib/omabit/hostcompat"
const VerificationReceiptPath = VerificationDirectory + "/receipts-v2.json"
const VerificationCacheDirectory = VerificationDirectory + "/evidence-v2"
const ReceiptValidity = 30 * 24 * time.Hour // diagnostic age only, not expiry

func ReadQualificationRecords(io.Reader) ([]QualificationRecord, error)
func EvaluateWithQualification(Report, string, []QualificationRecord) Report
```

`EngineVersionShell` generates the side-effect-free POSIX version predicate used
by reviewed rootless boot guards. `Inspect`/`InspectCurrent` execute nothing, write
nothing, and never fetch network evidence. Native root refresh is explicit or
invoked by a previously root-installed native post-transaction hook.

## Admission and qualification

Stable Docker **27.2.0 <= version < 30.0.0** is compatible after API **1.47**,
rootless, systemd cgroup v2, data-root and transport-identity checks. Installed and
running versions are independent: **two compatible versions plus verified native
packages remain usable**, with `restart_pending` as an advisory. An unsupported
or unresolved installed version still refuses new work. Safe observation/Stop
retain their independent API/containment checks regardless of package provenance.
Historical engine versions are not rewritten to the baseline. Stable SemVer
`+build` metadata, including Debian `+dfsg1`, is accepted by Go and the generated
shell predicate. Raw reported versions are preserved; installed/running precedence
comparison ignores build metadata. Prereleases and malformed strings are refused.

`29.9.1` remains compatible/unqualified. `Evaluate` alone never qualifies a host.
`EvaluateWithQualification` requires an explicit trusted record for a compatible
engine, an exact raw engine version, platform/distribution/profile, API range and required API, and the
complete exact package tuple (role/name/full version/executable paths). All native
roles must be represented; additional recorded roles are permitted and must also
match. Drift or partial evidence stays unqualified. A profile must identify the
attested running-daemon executable, kernel/systemd/launcher/configuration and
remaining host stack, not just a distro name. The caller must establish that exact
observed profile, including same-upstream native-revision daemon drift; installed
package versions alone do not establish running executable identity.

`QualificationRecord`/`QualificationPackage` are bounded JSON-tagged inputs in
`qualification.go`. The decoder accepts at most 256 KiB, 32 records, and 32 packages
per record. Load only reviewed embedded release data or a trusted release file;
remote host assertions cannot authorize qualification. **No tuple is invented
here.** The coordinator can populate records from actual RC12 or subsequent
qualification evidence when the complete package/profile/API tuple is available.
Future compatible versions such as 29.9.1 may qualify only with such an exact
record; without one they remain unqualified.

## Runtime reads: durable authorization, not OS-cache availability

The root-issued v2 receipt distinguishes:

* **Live requirements:** semantic native package identity/ownership, fixed trusted
  executable paths and current SHA-256, readable public native trust evidence,
  current APT source authority, and valid verification timestamp metadata.
* **Audit evidence:** original package metadata, OS archive/signature/index hashes,
  private trust/tool snapshots and retained authenticated proof. These paths are
  never opened by runtime receipt readers.

Consequently, `apt update`, OS archive-cache pruning, unreadable
`/etc/pacman.d/gnupg`, and unrelated dpkg status changes do not invalidate unchanged
authorized roles. Readers do not hash archives, APT lists, retained blobs, native
validation tools or the private GPG trust database. `InspectCurrent` uses a bounded
256-entry SHA-256 cache keyed by the actual kernel device/inode, size, mode,
UID/GID, link count, and nanosecond mtime/ctime. Every hit opens the current trusted
object and rechecks identity and all path/ownership bindings before and after
lookup. In-place tampering with restored mtime invalidates via ctime; replacement,
permission and owner changes also invalidate. Unknown filesystem identities are
not cached. No authorization decision or TTL is cached by the core. Metadata and
receipts are read anew on admission; an RPC layer may separately cache reports.

Each inspection has aggregate limits of **96 MiB metadata reads, 512 MiB hashing,
262,144 filesystem operations, 32,768 directory entries and 32,768 native package
records**. The existing 64 MiB native-database aggregate also applies. Budget
exhaustion reports unresolved evidence and cannot admit new work. Warm unchanged
inspections use identity checks rather than rereading large executable bodies.
Native ownership scans retain only the critical resolved executable paths.

Pacman public authority inputs are root-owned files in
`/usr/share/pacman/keyrings/` ending in `.gpg`, `-trusted`, `-revoked` (also the
equivalent dotted suffixes). APT authority inputs are canonical active root-owned
source definitions and their explicitly selected, runtime-readable public
keyrings. Public keyring/revocation/source changes invalidate authorization
immediately. **Direct edits to Pacman's private trust state must invoke the same
root refresh hook**: ordinary runtime users cannot inspect private trust changes.
**There is no wall-clock authorization expiry.** An unchanged authenticated host
continues to work after 90 days offline (and beyond). `VerifiedAt` and a historical
freshness diagnostic remain; zero, invalid and future timestamps are refused.
The old v2 `valid_until` field remains readable as legacy metadata, not an expiry
gate. `ReceiptValidity` is retained only as the diagnostic-age threshold for old
callers. Root refresh uses retained proof and current trust without network access.

## Root refresh and post-update hook contract

Call `VerifyCurrent(ctx)` locally as root after the native package transaction and
native keyring refresh complete, **before OS package archives are cleaned**. The
installer owns hook installation and invocation; this package does not install a
hook or change package-manager configuration. Hooks keep using **offline**
`VerifyCurrent`; an unchanged host requires no periodic refresh to stay authorized.

Concrete application-owned paths:

| Path | Ownership/mode | Purpose |
| --- | --- | --- |
| `/var/lib/omabit/hostcompat/` | root, 0755 when created | verification metadata |
| `receipts-v2.json` | root, 0644 | atomic runtime-readable authorization and audit index |
| `.refresh.lock` | root, 0600 | bounded serialized refresh lock |
| `evidence-v2/` | root, 0700 when created | private retained proof |
| `evidence-v2/<sha256>.<format>` | root, 0600 | content-addressed package/signature/InRelease/index blobs |
| `.fetch-<random>/` | root, 0700 (0711 for APT traversal) | optional acquisition parent; removed after explicit verification |
| `.fetch-<random>/apt/` | root, 0711; config files root 0644 | protected APT configuration; downloader cannot alter it |
| `apt/{state,cache,archives,lists,log,home,tmp}/` | existing `_apt`, 0700 | designated untrusted acquisition subtrees only |
| `.fetch-<random>/apt-evidence/{lists,archives}/` | root, 0700; files root 0600 | checked sealed copies for cryptographic verification |

Publication explicitly applies and verifies descriptor permissions before sync
and rename, so the native hook's `umask 077` cannot turn public receipts into 0600
files. Newly created public application parents are explicitly 0755; private
evidence remains 0700/0600. Recovery of an older 0700 public layout requires an
exact root-owned application subtree, a valid versioned receipt, and no unrelated
entries at public parent boundaries. Foreign owners, aliases and unrecognized
administrator trees are refused rather than widened; `/var` and `/var/lib` are
never chmodded by this helper.

Fresh package verification compares authenticated archive payload SHA-256 with
installed executable bytes, then copies **exactly those authenticated bytes** to
content-addressed blobs. Each blob and its parent directory are synced before
publication. The receipt is synced and atomically renamed only after proofs,
current semantic ownership and installed bytes have been checked. Concurrent
package/evidence/trust mutations fail closed. New receipts do not depend on later
availability of the original cache files.

Unchanged roles reuse retained proof under **current** native signature/trust
validation and current installed-byte checks, not an old `authorized` bit. APT
proof reuse may retain the historical Release expiry only for the exact previously
authenticated package; its signature must still pass current source authority,
revocation, key-expiry and strong-digest checks. Keyring filename rollovers can
revalidate proof from the same repository scope under its new configured keys.

Partial failure returns a report and error, withdraws authorization for failed
roles, and preserves their signed recovery proof. Trust races/unavailability
withdraw all prior authorization. Once the new receipt is durable, refresh prunes
only recognizable unreferenced blobs in its own cache; unauthorized recovery
proofs remain referenced. It never prunes an OS cache. There are at most eight
role receipts and three retained proof blobs per role (shared blobs deduplicate).
Older v1 receipts remain untouched and require explicit reauthentication into v2;
no persisted host/runtime state is reset.

## Native authentication and source scoping

Metadata preserves epochs and native revisions. Ownership/version metadata,
dpkg MD5 and recorded Pacman validation are not publisher authentication. A Pacman
recorded PGP check is explicitly labeled a **package-manager assertion**.

* **Pacman:** fixed native GnuPG with the Pacman keyring and PGP trust database,
  FULLY/ULTIMATE trust, strong signature digest and current revocation checks.
  Public native revoked-fingerprint lists are enforced even if a private trust DB
  has not yet incorporated a keyring-package revocation. No auto key lookup/import
  is allowed. Full package identity/architecture and bounded regular tar members
  are checked; `bsdtar` re-archives only to stdout, never to a filesystem directory.
  Local database directories and cached archives may include an epoch, e.g.
  `docker-1:29.7.2-1`. Both native archive naming forms are supported. If an adjacent
  `.sig` is absent, the matching repository database's base64 `PGPSIG` is decoded
  and independently verified against the exact archive/current trusted keys.
  Repository metadata/PGPSIG assertions alone never authenticate payloads.
* **APT:** root-owned `.list` and Deb822 `.sources` files bind each URI/suite/
  component/architecture scope to its own `Signed-By` keyrings. Normal official
  Docker source layouts work. `.gpg` binary and `.asc` armored file keyrings are
  supported; native `gpg --dearmor` converts armor without importing keys. GnuPG is
  therefore a required native validator for armored keys. Only that source's keys
  are supplied to `gpgv`; unrelated keys are not pooled. Native signature status
  is evaluated per `NEWSIG`: at least one strong valid selected-key signature is
  required. Additional missing-key signatures (matching `ERRSIG` no-public-key and
  `NO_PUBKEY`) do not defeat that signature, including native gpgv exit 2. Known
  bad/revoked/expired/weak selected signatures, malformed status and invalid-only
  input remain rejected. Only bounded gpgv status output is preserved on nonzero
  signature exits; other commands remain strict and stderr is discarded. InRelease
  cleartext is read from the captured, identity-bound clear-sign envelope after
  native validation, avoiding acceptance of partial plaintext from a second
  nonzero command. The chain is InRelease SHA-256 →
  Packages SHA-256 → `.deb` SHA-256 → installed executable SHA-256. A `.deb` is not
  assumed individually signed. Conventional distribution archive hosts without
  explicit Signed-By select only their corresponding native archive keyring.

`trusted=yes`/insecure bypasses and explicitly untrusted (`trusted=no`) sources
are never promoted to authority. Inline keys, fingerprint-only
Signed-By, unsupported key formats, credential-bearing/nonstandard URI layouts,
flat repositories, detached Release.gpg and conflicting source definitions are
reported as unresolved evidence. APT source definitions/keyrings must be readable
to the runtime UID, as with normal native `_apt` access.

Source fragments use native APT filename filtering: hidden files, disabled suffixes,
invalid filename characters and non-regular entries do not become authority.
Read-only inspection also reads the root-owned native `apt.conf.d`/`apt.conf`
configuration inputs. Effective `Dir::Etc`/root/source-selection/filter/architecture
overrides (including nested and binary-specific forms), `#include`/`#clear`, and
an invocation's `APT_CONFIG` are currently **unresolved**, never silently replaced
by the default source directories. This conservative supported-config boundary
requires no metadata command execution. The receipt binds a versioned default
selection model and active source-file set; root-trusted configuration hashes are
recorded for audit/race checks. Unrelated hook/cache/proxy settings are parsed as
data and do not revoke unchanged source authority merely by being added. Removing
or hiding a source, or introducing an effective selection override, withdraws its
authorization on both inspection and explicit root refresh.

Commands use fixed root-owned tool locations, fixed argv, a clean environment,
bounded output and process-group cancellation. Diagnostics never expose native
stderr or raw repository text. Refresh installs/upgrades no package, imports no
host trust key, probes no Docker engine and restarts no service. The default
`VerifyCurrent` is offline.

## Explicit fetch for initial setup/repair

OS-authorized setup/repair can call:

```go
report, err := VerifyCurrentWithOptions(ctx, VerificationOptions{FetchMissing: true})
```

Fetch is attempted only after local retained/cached proof cannot authenticate a
required installed package. It downloads **only matching installed package
builds**, without dependencies, installation, upgrade or key enrollment. Success
of a native download never grants provenance: the normal exact identity,
signature/index chain and installed executable SHA-256 verification still runs.
Missing old versions fail closed rather than falling back to a newer build.

* **APT:** native `/usr/bin/apt-get -qq update` prepares indexes under the unique
  private stage, followed by `apt-get -qq download -- NAME:ARCH=FULL_VERSION`.
  **Both commands run as the existing non-root `_apt` UID/GID with cleared
  supplementary groups**, and retain `APT::Sandbox::User "_apt"`. There is no root
  acquisition fallback and no account creation. Missing/unsafe local `_apt`
  identity is explicitly unavailable. Setuid/setgid fetch executables are refused.
  `APT_CONFIG` is loaded before normal apt.conf configuration and redirects main,
  parts, sources, state/lists, cache, archives and logs into that stage. Global
  post-update hooks/config overrides are excluded. Canonical root-authorized
  sources carry their exact `Signed-By` keyrings; insecure repositories, weak
  validation and TLS verification bypasses are disabled. This is a private index
  acquisition, **not** an update of `/var/lib/apt/lists` or installed packages.
  The staging monitor accepts `_apt` ownership only in the enumerated acquisition
  subtrees, not protected configuration, evidence or unrelated paths. After each
  child exits, expected indexes/archives are copied through pinned-directory fds
  and `O_NOFOLLOW|O_NONBLOCK`. Symlinks, hardlinks, unexpected owners, oversized
  files and mutation races are refused. Stable bytes are SHA-256 hashed, synced,
  atomically published and rehashed in root-only `apt-evidence` directories; only
  these copies reach normal signature/index/payload verification. Ownership
  promotion alone does not authenticate anything. Source keys and any native auth
  configuration must be readable by `_apt`; root-only credentials are not exposed
  or permissions broadened automatically. Raw errors never leave the verifier.
* **Pacman:** `/usr/bin/pacman-conf` reads the bounded root-owned configuration and
  include closure, including inherited signature policy and expanded servers.
  Only required/trusted package signature policy is accepted. Native
  `/usr/bin/curl` downloads private repository DBs and the exact matching archive
  selected by full name/version/architecture. Its `PGPSIG`, `.PKGINFO`, archive
  hash and installed-byte checks remain mandatory. Neither `pacman -Sy` nor a
  package transaction, external `XferCommand`, hook or key import is invoked.

Native fetch uses a clean environment/private home and temporary directory,
fixed argv, no shell, root-owned fixed executables and `/usr/bin/prlimit` file-size
limits. Protocols are restricted to HTTP(S); aliases/traversal and credential URLs
are refused. A staging monitor limits aggregate data (2 GiB threshold, plus bounded
in-flight writes), file count and depth. Artifacts are retained atomically before
the private stage is removed. Curl has a 60-second transfer limit; native fetch
commands have a two-minute limit and explicit fetch verification a ten-minute
total deadline (a shorter caller context still wins).

Required download tools are `apt-get` or `pacman-conf` + `curl`, plus `prlimit`;
existing cryptographic/archive validators are still required. No helper packages
are installed by verification. Current configured repositories must retain the
exact installed version. Custom global proxy/XferCommand hooks and unsupported
source/key formats are not adopted into the private downloader.

## Bounds and tests

Other per-object bounds include 16,384 directory entries, sixteen symlink hops,
256 MiB executables, **512 MiB uncompressed repository indexes** and package/tar
streams, 1 MiB receipts/engine responses, 64 source scopes, five seconds per engine
probe, twenty seconds per native verification command and two minutes per offline
refresh. Explicit verification additionally caps metadata at 256 MiB, file reads
and hash reads at 2 GiB each, native command output at 4 GiB, directory entries at
262,144, filesystem operations at 2,097,152 and parsed records at 1,048,576.

APT Packages data is streamed through SHA-256 and a one-stanza parser, including
large Ubuntu universe indexes. It retains only exact installed
name/version/architecture targets, not every version or description. Each unchanged
index is decoded once for the required stack, with at most 262,144 stanzas per
index and 64 target matches. Authenticated package tar members are hashed directly from bounded native output,
without retaining the expanded archive in memory. Trailing output counts toward
the aggregate limit, and decoder failure discards partial hashes without buffered
retry. Shared authenticated payload hashes are reused within one refresh. No repository-sized slice of paragraph maps is built.

Tests exercise real isolated GnuPG signatures, trust, revocation and armor
conversion; real Pacman archive conversion; source isolation; non-root-equivalent
receipt reads with forbidden private/cache paths; cache prune/APT list rotation;
post-update unchanged-role reuse and changed-package retention; payload/receipt/
proof tampering and mutation races; real temporary-filesystem atomic publication
and pruning; 90-day offline operation and invalid timestamps; exact future-version
qualification tuples; SemVer build metadata; API and mixed-version admission.
An actual Arch bootstrap fixture uses native pacman-conf/curl against an isolated
local HTTP repository, then real PGPSIG/installed-payload verification. APT fetch
argv/private state is injected, with real signed-index verification; native
`apt-get`, `dpkg-deb` and `apt-helper` execution still needs Debian/Ubuntu platform
integration testing because those tools are absent on this Pacman host. Focused
tests also exercise the InspectCurrent path's warm cache and same-inode,
same-size/restored-mtime tampering; aggregate budgets; a streaming index larger
than installed-metadata limits; `_apt` credential/path scoping; and real temporary
filesystem promotion with alias, mutation and cancellation failures.

Publication regressions include restrictive-umask creation/recovery, preserved
administrator trees, and a real root issuer → UID-1000 reader process test. The
latter can run inside a disposable mapped user namespace without host-root changes:

```sh
unshare --user --map-auto --map-root-user --setuid 0 --setgid 0 \
  go test ./internal/hostcompat -run '^TestRootHookPublicationReadableByUID1000$' -count=1 -v
```

Native multisigner fixtures cover one selected valid key plus two unavailable
signers, invalid-only input, selected-key revocation alongside another good signer,
and bounded nonzero-status handling. These tests bind the current source tree;
they do not qualify older VM candidate binaries or replace rerunning the native
Debian/Ubuntu transaction harness against a rebuilt candidate.

Runtime engine probes share a 15-second total read budget for `/version` and
`/v1.47/info`; a shorter caller deadline takes precedence. Boot-time containerd
latency must not be mistaken for incompatible engine metadata by a separate
2-second header limit. Every probe still checks socket ownership/peer credentials,
API compatibility, rootless/cgroup configuration, data root and engine identity.
Timeout/cancellation identity is preserved without exposing response bodies or
arbitrary transport diagnostics. This does not retry a runtime mutation.
