# Development loom and native control

RC.10 adds a loom-size dropdown and owner-controlled native access for every
ship class: fake, comet, moon, planet, star and galaxy. It does not automatically
update installed runners or change existing running ships.
The managed-fake gate passed for loom34, native file threads, a Clay byte round-trip
and explicit stop/start. See the [verification report](runtime-settings-validation.md).
Both the initial fake-only update and the all-ship successor were installed locally
after separate approval, preserving existing ship settings and runtimes. The new
preview exposes livenet opt-in, but no existing livenet socket was enabled or tested.
RC.10 publishes matching desktop/tool pins and final-binary fake/comet native
execution plus repeated RC.9 upgrade evidence. See the
[RC.10 qualification](diagnostics/2026-10-18-rc10-qualification.md); the
[contributor report](runtime-settings-validation.md) records earlier iterations.

## Settings and limits

| Setting | Default | Supported values |
| --- | --- | --- |
| Loom byte exponent | 31 | 20 through 34, inclusive |
| Native control socket | Off | Explicit owner opt-in for any ship class |

Loom size is `2^exponent` bytes of address space. Examples:

| Exponent | Address space |
| --- | --- |
| 20 | 1 MiB |
| 30 | 1 GiB |
| 31 | 2 GiB |
| 32 | 4 GiB |
| 33 | 8 GiB |
| 34 | 16 GiB |

These bounds come from the pinned Vere `4.6-5a7614f` parser and allocator.
They are parser limits, not a promise that every pill fits in every loom.
Small looms can fail to boot. Reducing a populated pier's loom can also fail.

**Address space is not resident memory (RSS) or a container memory allowance.**
The existing container cap remains 4 GiB, with no additional swap allowance.
The CPU and PID limits remain two CPUs and 256 PIDs. Startup admission still
requires at least 5 GiB available host RAM and 8 GiB free disk.
Check host available RAM and the separate container resource measurements before
starting a workload. Loom34 does not make a memory-heavy workload fit under the cap.

Common configuration validates the same loom range for every ship class. Native
control is an owner decision, not a ship-class restriction. The runner must advertise
`native-control-all-ships` to enable it on nonfake ships. Older `runtime-settings`
runners retain their fake-only behavior. No new livenet execution qualification
accompanies this successor.

## CLI

Use a collector built from this source and an explicitly selected runner.
The runner must advertise `runtime-settings` in `hello.features`. Nonfake native
opt-in also requires `native-control-all-ships`. Older runners receive an
update-required error before unsupported settings are submitted.
`create-comet` and `create-ship` also accept `--loom` and `--native-control`.

The following example registers a **new** development fake. It does not start it.
Choose an unused instance name and an available HTTP port first. Replace
`RUNNER_ID` below with the identity returned by `hello`.

```bash
urbit-collector --runner local hello
urbit-collector --runner local --expected-runner-id RUNNER_ID suggest-port
urbit-collector --runner local create-fake dev-native \
  --http-port 18090 --request-id create-dev-native-1 \
  --loom 34 --native-control --bootstrap bare
urbit-collector --runner local --format text snapshot
```

Omit both new options for the existing defaults: loom31 and native control off.
Content-hashed supplied pills remain available through
`--bootstrap recommended --pill URL_OR_FILE`. Initialized starts never reapply a pill.

Read the instance's current revision from a fresh snapshot before every operation:

```bash
urbit-collector --runner local operate dev-native start \
  --revision REVISION --request-id start-dev-native-1
```

Replace `REVISION` with the observed revision. Verify the returned operation ID in
subsequent snapshots. An accepted operation is not proof of a successful start.
Require the exact operation to succeed and the fixture to become running/healthy.

To change settings, send the complete desired configuration with its revision:

```bash
urbit-collector --runner local configure dev-native --revision REVISION \
  --http-port 18090 --ames-port 34343 --loom 34 --native-control=true
```

`configure` preserves the current native-control setting when `--native-control`
is omitted, including requests from RC.9 collectors. Specify `=true` when enabling
it, or `=false` when disabling it. These flags do not restart a
running ship. Apply changes with an explicit Omabit Stop followed by Start.
Use a fresh revision and a new request ID for each operation.

Text and JSON snapshots distinguish requested settings from the recorded launch.
JSON fields are `config.loom`, `config.native_control`,
`runtime_status.launch_loom` and `runtime_status.launch_native_control`.
Recorded launch values come from persisted container intent, not desired settings.
They are not live measurements. Check observed state, container identity and freshness.

## Desktop

The forms provide a **Loom address space** dropdown with all 15 supported exponents
and calculated MiB/GiB labels. **Enable local-owner native control** appears for
every ship class and defaults off. Owned-key creation includes these settings under
Advanced configuration. An older fake-only backend disables nonfake opt-in with
explicit upgrade guidance, rather than presenting it as a product-policy ban.

For a registered ship, open **Connectivity & runtime → Edit runtime settings**.
The editor preserves native control during an HTTP-only edit. Saving checks the
captured revision. Requested and recorded settings remain distinct until replacement.
Controls explain when the selected runner requires an update.

## Local native access

**Primarily for development. Leave disabled for routine use: it is unnecessary and adds risk.**

The pinned Vere build has a known ship-crash bug when a client disconnects with
replies pending. Keep clients connected until every reply has been received.
Do not kill clients or use short timeouts.

**`conn.sock` grants full ship-control authority. It is not a health-only interface.**
All processes with the Unix owner's authority can use it. Management observer roles
do not isolate programs already running as that owner. No TCP socket, remote shell,
extra bind mount, Docker-group grant or privileged container is added.

Enabled launches omit `--no-conn`. A fixed launcher sets `umask 077` before
executing Vere, so socket creation has no group/other permissions. Containment
checks require that exact launcher and the unchanged container restrictions.
Disabled launches retain the direct Vere command and `--no-conn`.

The snapshot supplies `runtime_status.native_control_socket` when its recorded
launch enabled native control. The path resolves the registered storage root,
including custom locations. It is local to the **runner host**, not the collector.
A displayed path or existing socket file does not prove that a listener is alive.
Verify owner, private permissions, current container and fixture identity first.

### Undocked file-thread procedure — tested on a managed fake

Omabit retains `--no-dock`, so a managed pier has no guaranteed host-executable
`.run`. Do not create or replace `.run`, and do not change the managed runtime.
Upstream `click` supports `-b PATH_TO_VERE` for an explicit local noun codec:

```bash
click -b /absolute/path/to/verified-vere-codec \
  -k -i /absolute/path/to/thread.hoon /absolute/path/to/owned-fixture-pier
```

The codec performs local `eval` serialization. It does not boot or manage the pier.
Verify its provenance and its jam/cue behavior before use. Do not assume that a
container executable can run directly on the host. Use a `click` build whose help
and source include `-b` and `-i`. A plain `click -k -i FILE PIER` is insufficient
when it expects a missing or incompatible `.run`.

The tested codec was copied from `/bin/urbit` in the exact new fixture container
using the existing owner-private Docker socket. This copied an executable, not a
pier, and did not change the container. Its `-R` result was `4.6-5a7614f`.
The copied codec ran directly on this Linux x86-64 host. Other hosts must verify
compatibility before use. No Docker-group grant or global codec install is required.

The test used upstream click and click-format at commit
`d454e2482c3d4820d37db6d5625a6d40db975864`. The operator must bind the container ID
to the selected fresh fake before copying its codec. Host-side file threads returned
`[0 %avow 0 %noun %omabit-native 0 42]` and exact Clay source bytes.
Retain unfiltered typed replies and stderr. Check the decoded success/result noun,
not the CLI exit code. In particular, upstream click can exit zero after a failed pipeline.

### Transport risk

Pinned Vere predates the disconnect fix merged in
[urbit/vere#1101](https://github.com/urbit/vere/pull/1101).
This build has a known crash bug when a client disconnects with replies pending,
even for read-only work. Treat that condition as unsafe, not as an acceptable gamble.
Do not use a short killed-click timeout, pipeline multiple requests, or run a crash
reproducer. Keep consuming a complete reply before closing. Prefer small responses.
The tested click script uses `nc -W 1`, a receive-packet limit, not a framed-reply
decoder. Qualification covers serial small replies only. Larger replies need a
frame-aware draining client and separate verification. A drained reply reduces
exposure but is not a guarantee against transport failure.
Preserve raw evidence and report uncertainty if transport or decoding fails.

## Persistence and upgrade

Schema19 upgrades schemas11–18 transactionally without rewriting their instance or
operation bodies. Missing private launch fields mean historical loom31 and native
off. Existing requested values, receipts, identifiers and piers remain intact.
Old runners refuse schema19 because they cannot preserve settings-aware handoff
contracts. Schema17 was the fake-only preview; schema18 broadened native opt-in.
Do not downgrade the database version, reset state or restore a pier copy as rollback.
A manager update alone neither restarts ships nor changes their recorded arguments.

Nondefault loom/native settings are preserved during cross-host moves and covered
by the transfer's signed commitment. Both runners must advertise
`runtime-settings-handoff`; the collector checks this before fencing the source.
Historical/default-setting transfers retain their original signed contract.

## Approval-gated managed acceptance

The current production entrypoint uses the installed owner profile, one fixed
management directory and one socket per host. It does not offer a private state-root
override. Do not bypass its lock or start a second manager against shared state.

On an already active host, first approve a reviewed management-only update. This
approval is separate from source work. The recorded run used a separately reviewed
binary-only replacement, not the broader host installer. Preserve all existing
running containers, settings and storage. Do not execute a fresh-install reset.
A separate disposable host with an approved owner installation is also suitable.

The acceptance operator must record and review these inputs before updating:

- Exact source/build hashes for runner, collector and desktop assets.
- Existing owner, runner identity and management-only update procedure.
- Exact update effects. If using `install-host`, review its bundle/key inputs and wider host-setup effects separately.
- Database migration/downgrade implications and assurance that no ship restart is requested.
- Unique fixture names, content-pinned pill choice, evidence directory and codec provenance.

After approval, use the following test sequence. No unrelated pier operation is allowed.

1. Read `hello` and require `runtime-settings` from the corrected runner.
2. Register a fresh loom34/native-enabled fake and a separate default-disabled fake.
3. Start each through Omabit with revision-bound, unique operation requests.
4. Record successful receipts, exact container IDs, image digest and actual process arguments.
5. Check loom34 on the worker process, not only the initial lightweight boot banner.
6. Check the unchanged cgroup cap independently from loom size and resident usage.
7. Require owner-private pier/socket permissions and a reachable enabled socket.
   Require no socket on the fresh default-disabled fake. Inspect its `--no-conn` argument.
8. Execute a small file-based native thread with a known result, such as a tagged integer42.
   Preserve the complete typed reply and compare the decoded tag and value.
9. Read the new fake's actual hood generator definitions before constructing desk operations.
   Create and mount a fresh test desk through native threads. No HTTP substitute is accepted.
10. Copy a small Hoon file through ordinary owner access. Commit the desk natively.
    Scry its actual Clay bytes through `%cx` and compare them with the intended fixture bytes.
    Mount contents, an acknowledgement, or an exit code alone do not pass this check.
11. Change only the enabled fixture's desired settings. Confirm its old launch stays correctly labeled.
    Stop/start through Omabit and verify the replacement arguments and preserved desk bytes.
12. Stop both exact fixtures through Omabit. Verify each successful receipt, stopped state,
    removed runtime containers and absent owned runtime processes. Managed Stop
    retains the registered pier's dedicated bridge. Verify that it has no attached
    containers and record it as retained, not removed. Retain both stopped piers
    and all evidence. Do not delete them implicitly.

Stop command shape, with values from the current selected fixture:

```bash
urbit-collector --runner local --expected-runner-id RUNNER_ID \
  operate FIXTURE stop --revision REVISION --request-id UNIQUE_STOP_REQUEST
```

If native transport fails, stop issuing native requests. Preserve the raw failure
and use Omabit for owned lifecycle recovery. Do not kill arbitrary Vere processes.
These steps passed on the recorded fresh fakes, including Clay byte verification
after replacement. Repeat them for materially different runtime/client combinations.
No whole-urgit compatibility or broad security qualification claim follows.
