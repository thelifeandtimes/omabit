# Desktop releases, profiles and upgrade qualification

This describes the **RC.14** desktop architecture. Older RC.13 installations use
static wrappers; updating the native plugin installs the new bootstrap. Candidate
builds/imports and isolated qualification also work directly from this checkout.
No change to stock Omarchy is required.

For work after stable0.1.0, follow the [branch/version policy](development-policy.md).
`master` is release-only; use edge/develop or topic branches for iteration. Edge
artifacts use `-edge.N` names and explicit branch updates, and the UI labels the
distribution channel independently of the loaded artifact version.

## Architecture and compatibility

The native manifest selects `native/v1/Service.qml` and `native/v1/Panel.qml`, with
stock Omarchy's `keepLoaded:true`. The service is a small retained bootstrap; the
widget is a replaceable adapter. Neither statically imports the changing application.
Changing both entry-point URLs also gives existing RC.13 users a fresh bootstrap
type on their first upgrade, rather than reusing the old wrapper's cached type.

The paired UI and three tools come from one size/SHA-256-pinned release archive.
Preparation validates archive paths, members, expansion bounds, source/version
identity and executable pins. Completed generations are atomically published under:

```text
$XDG_CACHE_HOME/omabit/releases/<descriptor-digest>/
    ui/                 # QML, JavaScript, icons and manifest
    bin/                # matching omabit, collector and runner executables
    BUILD.json
    bundle.tar.gz       # retained checksum-pinned authentication input
    inventory.json
```

Every nested QML/JS path changes between releases. Cache verification binds UI bytes
to the retained pinned archive, not merely to an editable inventory file. Downloads
and state files live outside the watched plugin checkout. Each running UI receives
its own exact tool directory; it does not follow a mutable launcher to newer tools.

One lease-owning bootstrap may activate a desktop at a time. Selection, preparation,
durable activation intent and readiness acknowledgement are separate. The bootstrap
waits for setup/submissions/clipboard work and pending first boot to settle, freezes
UI admission, loads and checks the candidate service and widget, and waits for saved
enrollment and initial observation. An empty registry is valid readiness; offline
runners may report their own failure without making the whole desktop unavailable.

Failed candidates retain the last working release. A lost commit acknowledgement
can restore only that exact activation transaction's previous desktop generation.
Failed requests are not retried continuously; use Retry or select another release.
The diagnostic IPC distinguishes selected versus actually loaded version/profile.
Inventory reconnects use bounded exponential backoff and retain explicitly stale
observations until fresh observations arrive. No mutation is automatically replayed.
Profile preferences retain the selected monitor page and ship.

Desktop state is version1 in `$XDG_STATE_HOME/omabit/bootstrap/state.json`, protected
by a lock and atomic, synced writes. Application `DESKTOP.json` declares bootstrap
API1 and profile-state schema1. Unknown state/contracts are refused and retained.
Legacy archives without this contract can load only in the default profile; they
cannot claim isolation they do not implement. Bootstrap API1 must remain compatible
across application releases; an incompatible bootstrap needs a new entry-point/API
generation and a separately tested migration.

## Profiles

**Default** uses existing configuration, enrollment, credentials and preference
paths without relocating them. It follows the published plugin pins unless the
operator explicitly selects a release.

Named profiles have independent `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `XDG_DATA_HOME`
and `XDG_CACHE_HOME` beneath `.../omabit/bootstrap/profiles/<name>/`. Helpers receive
these roots on every invocation; QML preferences use the profile state root too.
Profiles do not inherit the default collector's credentials or runner enrollment.

Creating an existing profile is refused. Create another name for another clean
desktop run; previous state is retained. There is no profile/pier reset command.

Profiles isolate **desktop state**, not Linux accounts, installed runner services
or piers. A fresh host installation requires a fresh disposable VM. Local host
setup is disabled in named profiles unless created explicitly with `--host-setup`.
Runner commands/upgrades retain their normal identity, authentication, permissions
and migration contracts. Selecting an old desktop never downgrades a runner database.

## Daily development

Use a topic branch/worktree. A local iteration does not need a public version,
release PR or the entire release suite. Start with the narrowest useful check:

```sh
# UI/model/bootstrap feedback (no runner operations)
python3 -B scripts/omabit-dev.py check --scope desktop
# Packaging, cache, channel and release-policy feedback
python3 -B scripts/omabit-dev.py check --scope tooling
# Exact Go package/test feedback; omit --run to exercise the whole package
python3 -B scripts/omabit-dev.py check --scope go --package ./internal/client --run TestLocalImportPreviewByteBoundaries --race
# Actual Qt binding checks; requires the documented Qt dependencies
python3 -B scripts/omabit-dev.py check --scope qml
```

Plain `check` runs desktop and tooling checks. `--plan` prints commands. These
are feedback lanes, not a claim of full Go, race or live upgrade qualification.
Before promotion, run the complete CI and upgrade gates below. Check dependent
Go packages too when changing a shared interface; `--scope go` without `--package`
runs all packages and vet.

Build an immutable snapshot of tracked/unignored source, including local changes:

```sh
mkdir -p build
python3 -B scripts/omabit-dev.py build --output build/dev-01 --label review-01
python3 -B scripts/omabit-dev.py stage build/dev-01 --profile development
```

The output directory must be new. Builds record the actual base commit,
source-file hashes, snapshot digest, toolchain and truthful modified-source flag.
They produce complete paired UI/tools and refuse source drift during the build.
These development artifacts retain the base version with a visible development
label; they must never be uploaded as replacement stable artifacts.

Development builds reuse private Go module/compiler caches under
`$XDG_CACHE_HOME/omabit/development/go` (normally `~/.cache/omabit/development/go`).
Use `--cache-dir /absolute/private/path` to choose another cache or `--no-cache`
to diagnose a clean build. Symlinked/shared cache directories are refused. Module
contents are verified before publishing each snapshot. HOME, Go configuration,
telemetry, workspace and compiler flags remain controlled. Public release
packaging still uses fresh caches and its independent reproducibility gate.
Caches consume disk; inspect them with `du -sh` and remove only this generated
cache when no build is using it if space is needed. No automatic pruning occurs.

`stage` verifies/imports the complete archive, creates the named profile if
needed, and prepares it **without changing the active selection**. Repeating it
preserves the profile's enrollment, credentials and preferences. It refuses the
default profile and profiles with local host setup enabled. It prints the exact
`select ... --release DIGEST` command for manual activation; run that command
when ready. The retained bootstrap notices the selection and activates when safe.

A profile isolates desktop state, not runner hosts. Enroll only the intended test
runner. Older separately namespaced DEV installations using static wrappers do
not consume these bootstrap selections; see [native DEV review](native-dev-review.md).
A profile pin survives native plugin updates. Selection is a request: verify the
actually loaded generation in Setup and with the bootstrap's status command.

```sh
python3 -B scripts/omabit-dev.py status
omarchy shell thelifeandtimes.omabit.bootstrap status
python3 -B scripts/omabit-dev.py exec --profile development urbit-collector runners
# Retry a failed activation, or return to the published desktop:
python3 -B scripts/omabit-dev.py retry
python3 -B scripts/omabit-dev.py select default --release follow
```

**Setup** reports loaded/requested version, profile, digest, source/development
identity, collector readiness and activation errors. Failed activation retains the
last working generation. Returning to a desktop bundle does not undo runner schema
migrations. Upgrade host services separately and only through supported paths.

## Verification layers

1. **Fast source checks**: archive tampering, profile isolation, state fencing,
   competing leases, stale acknowledgements, interruption/rollback and guest guards.
2. **Real Quickshell lifecycle**: separate immutable QML/JS generations, paired
   tools, busy deferral, failed-candidate recovery and process restart, with inert
   fixtures and private HOME/XDG roots.
3. **Actual application bundle**: the same bootstrap loads the packaged application
   and real helpers into a fresh isolated profile. The stock Omarchy QML modules
   are read-only dependencies; a Wayland socket supplies PanelWindow support.
4. **Full stock-Omarchy VM scenarios**: actual native installation/update commands,
   optionally privileged setup, running/stopped fake ships and runner upgrades.

```sh
python3 -B scripts/test-desktop-bootstrap.py
python3 -B scripts/test-desktop-bootstrap-qml.py --require-quickshell
python3 -B scripts/test-desktop-bootstrap-qml.py --require-quickshell \
  --application-assets build/dev-01
python3 -B -m unittest discover -s scripts/qualification/desktop -p 'test_*.py'
python3 -B scripts/qualification/desktop/qualify.py plan
```

The actual-application check uses private writable roots and opens no windows. It
requires stock Omarchy modules and a Wayland session; the basic lifecycle test runs
offscreen. Neither uses the live desktop's IPC or performs a runner operation.
Executed results and remaining gates are in the
[development verification record](diagnostics/desktop-bootstrap-development.md).

See the [VM harness](../scripts/qualification/desktop/README.md) for clean-install,
upgrade and full-host scenarios. Run that matrix on both target Omarchy versions.
Selecting a release in a profile is useful development coverage but does not
replace running the actual `omarchy plugin update` command in those guests.

## Release procedure

Use the existing clean-source release packager for publication. Its archive already
contains the UI and matching tools. After publication, native pins must include the
archive's name, byte count and SHA-256 in `bundle`, alongside the three executable
pins. Keep the application manifest/UI version consistent with the bundled tools.
Qualify the migration from the previous installed bootstrap as well as successive
application generations; cold-start success alone is not upgrade acceptance.

Desktop rollback uses retained compatible UI/tool bundles. Runner downgrade rules
remain governed by the [upgrade policy](upgrade-policy.md): a schema21 database
cannot be reopened with a schema20 runner. Return a full test host to an older
baseline through a disposable VM snapshot containing its matching state and piers.
