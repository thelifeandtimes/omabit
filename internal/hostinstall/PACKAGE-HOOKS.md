# Native package authentication refresh

Normal hosting setup installs native transaction hooks **after** the new trusted
management executable has completed its ordinary quiescence-aware activation.
It never replaces `current/urbit-runner` early to obtain a verification command.
A compatible rollback keeps its previous hook state and activation semantics.

An offline-upgrade continuation that finds its new manager already live still
repairs/revalidates native hooks before reporting ready. A failure during hook
publication is therefore retryable after management release without restarting
the manager, reacquiring its update fence or changing accepted work.

## Interrupted first owner setup

New owner installations record format1 initial engine activation intent at
`/etc/urbit-runner/owner-engine-activation.json`, root-owned mode0600. The record
binds the complete owner profile and is published before `owner.json`. Its
`prepared` stage permits the same initial setup to retry lingering, user-manager
startup and `enable --now` if interrupted before engine readiness.

The record becomes `complete` after private-socket and API verification, before
management installation. Later failures/retries do not repeat engine startup.
An established profile without this newer record also never authorizes automatic
startup; a missing record is not evidence that an existing engine is disposable
or needs starting. Ambiguous older partial setups require explicit review/start.
Malformed, foreign-owner, unsafe or stale-prepared records conflicting with
established management state are preserved and refused. Publication is atomic and
restart-safe; this additive installer journal does not change managed state schema.

## Initial missing evidence and explicit download authorization

Normal OS-authorized setup/update first inspects current authentication evidence.
If refresh is needed, it calls `hostcompat.VerifyCurrentWithOptions` with
`VerificationOptions{FetchMissing:true}`. This permits native package
**download-only** retrieval into core's private verification cache so preinstalled
packages need not already have archives in the OS cache. Core remains responsible
for source policy, signatures/hash chains, private-cache permissions, public
receipts and matching installed executable bytes. No package installation/update,
repository editing or key import is authorized by this option.

Verification completes before executing Docker or making service changes. Failed
downloads/authentication leave those actions undispatched; private verification
cache material may remain for a later attempt. Already verified current receipts
avoid retrieval. The historical offline management-only bridge retains its
separate upgrade contract.

`RepairEngineGuard(ctx)` remains offline. The explicit repair seam is:

```go
RepairEngineGuardWithOptions(ctx, VerifyOptions{FetchMissing: true})
```

The local CLI exposes that opt-in as `repair-engine-guard --fetch-missing` together
with its normal root authorization. The zero-value options retain offline
verification and the same guard-only, no-engine-restart behavior. Native
transaction hooks continue calling the offline `verify-host-packages` form below;
they do not gain download permission from setup's authorization.

## Installed files and invocation

| Provider | Native hook |
| --- | --- |
| Pacman | `/etc/pacman.d/hooks/00-omabit-host-packages.hook` |
| Debian/Ubuntu APT | `/etc/apt/apt.conf.d/00-omabit-host-packages` |

Both call the root-owned, fixed wrapper:

`/usr/local/lib/urbit-runner/package-hooks/refresh-host-packages`

The wrapper invokes only:

```text
/usr/local/lib/urbit-runner/current/urbit-runner verify-host-packages --acknowledge-local-verification --format json
```

It requires local root, checks trusted directories and the regular, exclusively
linked current executable, scrubs the child environment, ignores stdin, refuses
arguments, and applies a 130-second deadline with a five-second kill grace. Core
verification also imposes its own two-minute deadline and bounded local parsers
and tools. Only the verification subprocess and its children are timeout targets.

Verification failure prints a fixed diagnostic and **does not fail the OS
transaction**. The wrapper's success exit code is deliberately not evidence of
authentication. Only core's independently validated package receipts confer
provenance. No hook installs/upgrades packages, fetches keys, changes repositories,
restarts Docker/management, or touches piers. Existing running work is preserved;
an authentication failure can still prevent new runtime work under core policy.

## Ordering and supported configuration

Pacman refresh runs after the transaction and before conventional cache-cleanup
hooks. The installer asks fixed `/usr/bin/pacman-conf HookDir` for effective
configuration. It requires the normal local hook directory and refuses earlier
post-transaction hooks whose effects have not been reviewed. It does not mutate
`pacman.conf` or existing hooks.

APT refresh uses an early `DPkg::Post-Invoke` configuration fragment. The installer
reads effective configuration with fixed `/usr/bin/apt-config dump` and requires
its refresh to be the **first** post-invoke command after publication. Thus normal
`docker-clean` and similar later cache-cleanup fragments run after verification.
Ordering includes both anonymous and named direct children (for example,
`DPkg::Post-Invoke::cleanup`), in native tree order. Empty child values and nested
grandchildren are not executable entries under native APT `RunScripts` semantics.
If configuration clears/reorders that list, setup reports the unresolved ordering
instead of claiming working integration. Files are preserved for review. No APT
pre-install package-list input is accepted or interpreted.

The phase choice was checked against upstream APT2.6.1: `pkgDPkgPM::Go` runs
[`DPkg::Post-Invoke` before returning](https://github.com/Debian/apt/blob/2.6.1/apt-pkg/deb/dpkgpm.cc),
and [`InstallPackages`](https://github.com/Debian/apt/blob/2.6.1/apt-private/private-install.cc)
performs `APT::Keep-Downloaded-Packages=false` archive cleanup only after
`DoInstall` completes. Early post-invoke refresh therefore precedes both that
built-in cleanup and later configured cleanup hooks. This source review does not
replace disposable-host native transaction qualification.

The core verifier must durably retain authenticated evidence during this refresh,
before later native cache cleanup. Native `CacheDir`, APT cache/source aliases,
per-source `Signed-By`, and durable proof invalidation belong to `hostcompat`, not
shell parsing. Hook installation supports the standard configuration directories;
custom hook directories, alternate APT roots, earlier cleanup, and direct `dpkg`
invocations need separately reviewed integration. Unknown providers refuse hook
installation. This is source/fixture coverage, not live distro qualification.

Until core exposes nondefault native-cache resolution, setup also checks effective
Pacman `CacheDir` and APT `Dir::Cache::archives` against the verifier's standard
cache locations. Custom cache locations are explicitly refused, rather than
installing a hook that silently cannot discover the next transaction's packages.
Equivalent relative/absolute default APT cache configuration is accepted. Cache
paths never become wrapper arguments or shell commands.

All installed files and parents must be root-owned without symlinks or broad write
permissions. Repeated installation accepts identical reviewed contents. Exact
previous generated versions can be added to the compiled migration allowlist;
format1 is the first version and has no invented predecessors. Administrator
edits are refused before either wrapper or native hook is replaced.

## Existing-host upgrade, without reset

1. Use the newly obtained OS-authorized runner to update the existing management
   installation. It revalidates native package authentication, migrates only exact
   known29.7.2 guard/unit contents, and keeps the pinned launcher path, owner,
   registered state, custom pier root and engine data root.
2. Management activation retains the existing quiescence/fence/journal and
   schema-aware rollback rules. After successful activation, the installer verifies
   the installed executable matches the authorized build and publishes hooks.
3. If an OS update already stopped the engine under the old exact-version guard,
   run the new local authorized `repair-engine-guard` command first. It refreshes
   package authentication and changes the known guard without starting the engine.
   Complete an explicit coordinated engine start/restart, then finish the normal
   management update. There is no database/pier reset or implicit engine restart.
4. Subsequent supported native transactions refresh evidence automatically. A
   package/running-daemon mismatch still follows runtime restart-pending policy;
   hooks never hide it by restarting the daemon or rewriting runtime history.

For historical managers that require `install-host --offline-upgrade`, retain
that existing management-only preparation/locking procedure. Its successful new
manager activation installs hooks, but does not silently expand the bridge into
an engine update. Follow with the new local `repair-engine-guard` command for
package authentication and exact-content guard migration, then any explicitly
coordinated engine start/restart. Existing database schema fences and rollback
refusals remain authoritative throughout.

The development-only `install-rootless` helper does not install a management
executable, so it cannot install hooks. Complete `install-host` to obtain the
managed native-transaction integration. Existing installations acquire hooks by
repeating the supported management installer; guard-only repair stays guard-only.
