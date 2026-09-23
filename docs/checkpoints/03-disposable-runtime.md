# Checkpoint 2B: disposable Vere lifecycle

Prerequisite: checkpoint 2A has passed. On the current host, the dedicated account
is UID/GID 959, but the program discovers those IDs; they are not hardcoded.

This checkpoint runs **real containers and a fresh fake Urbit**, not the earlier
simulator. It uses no existing pier, real identity, bootstrap key, Dojo command,
login code, or Vere control socket. It does not change the Docker installation.

Status: implementation and fixture tests complete; manual runtime acceptance is
pending. Follow this gate before moving to the production runner implementation.

## Scope

The command performs this sequence:

1. Verify the dedicated account, private socket, engine identity, and capacity.
2. Acquire a per-account qualification lock and check unresolved prior runs.
3. Pull and authenticate a single immutable `linux/amd64` image candidate.
4. Download the upstream default pill, verify its exact size and SHA-256, and
   store it read-only in fresh account-private storage.
5. Start an isolated probe container, read actual cgroup limits, and execute
   `urbit -R` to check the binary version.
6. Boot a fresh fake `~zod`, observe private HTTP reachability, and stop cleanly.
7. Start a replacement container against that same new pier, check it, and stop.
8. Run offline `pack`, then start/check/stop another replacement container.
9. Remove the run's authenticated containers; retain its report, pill, and pier.

All containers have `--network none`, no published ports, no added capabilities,
no automatic restart, and bounded logs. Normal Vere invocations include `-L`,
`--no-conn`, and `--no-dock`. Initial creation uses `-F zod`; later starts omit
creation/bootstrap flags. The verified URL pill is mounted read-only and passed
through `-B`, so an unavailable or changed upstream object never silently becomes
a different boot artifact.

HTTP readiness uses Eyre's public `/~/host` endpoint from inside the container;
response bodies and headers are discarded. It proves HTTP reachability, not desk
inventory or application health. There is no browser port on the host to visit.

Offline pack is invoked as `urbit pack --loom 31 /urbit/fake-zod`, only after all
preceding containers have been stopped and removed. Meld is a later gate.

## Candidate and provenance

| Item | Pin |
| --- | --- |
| Image repository | `docker.io/tloncorp/vere` |
| Image manifest | `sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd` |
| Image config | `sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245` |
| Vere binary | `4.6-5a7614f`, verified at execution time |
| Default pill URL | `https://bootstrap.urbit.org/urbit-v4.6.pill` |
| Pinned object URL | `https://bootstrap.urbit.org/urbit-v4.6.pill?generation=1790171686269740` |
| Pill bytes | `217421172` (approximately 207.3 MiB) |
| Pill SHA-256 | `f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b` |

The SHA-256 was computed from the complete HTTPS object on September 23, 2026.
It is an observed-content pin, not an independently signed publisher checksum.
The version-shaped pill URL is mutable. If its content changes, verification must
fail until a new candidate is reviewed; do not bypass or automatically replace
the pin. The default object changed during development; the current candidate is
the September 23 object generation shown above. An older generation was no longer
publicly retrievable. The qualifier requests the selected generation explicitly;
if upstream removes it, fail and review a replacement rather than following the
mutable default URL automatically.

The image manifest/config were inspected through the registry, and bounded binary
layer inspection found the expected version. Runtime `-R` remains necessary.
Docker 29's containerd backend may report the manifest digest as its image ID;
legacy backends report the config digest. The qualifier authenticates the fixed
repository digest/platform, accepts only these known IDs, and records the actual
backend ID used for all container operations.

This is the upstream distribution, not the unreleased Omabit pill. Expected
base/webterm/groups desks still need separate application-level validation.

## Resources and time

- Admission requires **5 GiB currently available RAM** and **8 GiB free disk**.
- Each container is limited to **4 GiB memory, no additional swap, two CPUs, and
  256 PIDs**. The probe reads the live cgroup files before any ship boots.
- The ship uses `--loom 31`, a 2 GiB loom. Loom capacity and container memory are
  different limits; neither value is inferred from process RSS.
- The pill download is capped at 256 MiB and five minutes. The image pull, each
  HTTP readiness phase, and offline pack each have a ten-minute deadline.
- Normal shutdown sends SIGTERM and allows up to 300 seconds before Docker's stop
  timeout. A forced kill, OOM, or nonzero Vere exit fails the checkpoint.
- Cleanup has a separate bounded recovery budget, up to eleven minutes for each
  unresolved container intent. Usually containers stop promptly; this reserve
  covers a Docker request still finishing after its client was interrupted.

Admission is a snapshot, not a resource reservation. Close memory-heavy work if
needed. There is no automatic enlargement of limits or retry of interrupted pack.
Image cache and retained fake-pier data consume disk beyond the pill itself; the
free-space check is not a filesystem quota.

## 1. Build and review the plan

From the repository root as your ordinary desktop user:

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...
./build/urbit-runner runtime-plan
```

Expected: the pins above, `mode: disposable-fake-only`, disabled networking/control
socket, fixed resource requirements, and `runtime_qualified: false`.

`runtime-plan` does not contact Docker, download anything, or write files. The
qualifier accepts no alternate image, pier, identity, endpoint, or shell command.

## 2. Make the development binary accessible to the service account

The dedicated account should not need access to your home directory or checkout.
Copy the reviewed build into a root-owned checkpoint location:

```bash
sudo install -D -o root -g root -m 0755 ./build/urbit-runner /usr/local/lib/urbit-runner/checkpoints/2b/urbit-runner
```

This installs a development executable, not a new service. Do not replace it while
a qualification run is active. Future packaged releases will provide this binary
without a manual copy step.

After a code fix, rebuild and repeat this copy: rebuilding `./build/urbit-runner`
alone does not update the executable used by the dedicated account.

## 3. Run as the dedicated account

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/2b/urbit-runner \
  qualify-runtime --acknowledge-disposable-run
```

The outer sudo authorizes switching accounts. **The qualifier itself runs as
`urbit-runner`, not root.** Calling `sudo ./build/urbit-runner qualify-runtime`
directly will be refused, as will running it as the desktop user.

Docker adds a sticky bit to its runtime socket to protect it from periodic
cleanup. Mode `1660` (`0660` plus that bit) is expected and accepted, as is `1600`.
The qualifier still requires the dedicated UID/GID, a private `0700` runtime
directory, and no public socket access. Do not strip the sticky bit or change
permissions to work around an older qualifier's exact-mode rejection.

Progress appears as JSON lines on stderr:

```json
{"phase":"engine"}
{"phase":"image"}
{"phase":"pill"}
{"phase":"probe"}
{"phase":"boot"}
{"phase":"restart"}
{"phase":"pack"}
{"phase":"post-pack"}
{"phase":"cleanup"}
```

Image/pill provisioning and the first boot can take several minutes without an
additional progress line. Leave the terminal open. No raw Vere logs, cookies, or
login codes are printed.

## 4. Inspect the final result

The final JSON report is on stdout. Important successful fields are:

```json
{
  "checkpoint_passed": true,
  "runtime_qualified": false,
  "cleanup_complete": true,
  "limits_observed": true,
  "pier_metadata_seen": true
}
```

The complete report also includes candidate pins, the observed backend image ID,
phase outcomes, exact container names/IDs, and the retained directory/report path.
Every container should have `removed: true` and `create_uncertain: false`. All
phases should pass. Candidate fields identify the intended artifacts; their
presence alone is not proof that the corresponding phase passed.

`runtime_qualified` intentionally remains false. This gate does not prove Ames,
host port forwarding, expected desks, application state semantics, meld, reboot,
control-socket safety, or production failure recovery.

Reports are written at mode 0600 beneath:

```text
/var/lib/urbit-runner/.local/share/urbit-runner/qualification/<run-id>/report.json
```

Each run uses a new private directory and new fake pier. Successful runs leave no
qualification containers running, but retain the image cache, verified pill,
fake-pier data, and report. There is no image/volume prune or recursive deletion.

## Cancellation or failure

Ctrl+C requests cancellation and bounded cleanup. It does not simply abandon a
Docker-managed process. Allow cleanup to finish; do not repeatedly interrupt or
force-kill the qualifier unless necessary.

On failure, the command exits 1 and emits the report plus a JSON error. If failure
occurred before a run directory was created, its directory/report fields are
empty. A normal argument/acknowledgement error exits 2 without starting work.

If it refuses before any progress phase and `run_id` is empty, no Docker workload
was created. In that case `cleanup_complete: false` means cleanup was not needed
or attempted, not that a container was left behind. Correct the reported
precondition and retry without deleting runtime state.

If `cleanup_complete` is false, or a container has `create_uncertain: true`, stop
and share the report. An empty `docker ps` result is not proof that an accepted
create request has finished; it can become visible later. Do not edit report
flags, delete the report/lock, or run broad `docker rm`, `docker prune`, or
`docker system prune` commands to force another attempt.

Subsequent runs reject unresolved reports or leftover qualification containers.
Only bounded report metadata is examined; old pier contents are never traversed
or adopted. At most 256 prior runs are considered. Ambiguous outcomes require
manual reconciliation; an automatic recovery service is outside this checkpoint.

Even if cleanup succeeds, report a failed boot/pack before retrying. A rerun uses
a different pier and does not resume an interrupted maintenance operation.

## What to send back

Send the final JSON report and any `qualification_error` output. In particular:

- Did the engine/image/pill/probe phases pass?
- Did boot, replacement restart, pack, and post-pack restart pass?
- Are all containers removed and all outcomes resolved?
- Did timing or resource usage noticeably disrupt the workstation?

Do not supply login codes, real keys, or the contents of a real pier. We will
record this result before expanding to networking, desk checks, or the real
runner's durable lifecycle API.

## Developer verification and references

Fixture tests exercise both Docker image-store identity models, the installed
private `0660` socket, live-limit observations, pins, command ordering,
containment, bad inspections, OOM/nonzero exits, cancellation, delayed create/start
completion, bounded cleanup, and unresolved prior-run gates. No test runs Docker,
creates namespaces, or boots a ship.

The full upstream pill was streamed to compute the recorded SHA-256, but no pill
or image has been executed by the developer. Manual runtime validation remains
the purpose of this checkpoint.

- [Pinned image metadata](https://hub.docker.com/v2/repositories/tloncorp/vere/tags/latest) — the tag can move; use the digest above.
- [Vere source for the embedded build](https://github.com/urbit/vere/tree/5a7614f69bbc783736b08965fdc0a2534bb42e82)
- [Containerd image-ID semantics](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/daemon/containerd/image_inspect.go)
- [Open control-socket issue](https://github.com/urbit/vere/issues/1100)
- [Proposed control-socket fix](https://github.com/urbit/vere/pull/1101)
