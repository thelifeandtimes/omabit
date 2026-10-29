# Private chain qualification

This opt-in fixture compiles pinned public Bridge/Azimuth contracts, reproduces
Urbit's six local contract addresses, and deploys/reopens a disposable Ganache
chain. The `native-peers` stage additionally boots two fresh keyed test ships and
checks application recovery. This is qualification tooling, not a production chain
launcher or Docker devnet. No host network, production key or installed plugin is
accessed.

Use `inputs.json` to acquire the exact public source files into a private source
directory. Stage the included package/lock files in a separate tools directory
and use `npm ci --ignore-scripts --no-audit --no-fund` there. Do not execute
upstream pilot/install/cleanup scripts. The runner fetches nothing. Native npm
scripts are disabled; the qualified Node26 run used Ganache's JS fallback.

Run `python3 -B run.py --stage compile --sources SOURCE --tools TOOLS --output
/tmp/omabit-private-chain-compile-NAME`, then `--stage deploy --compiled
COMPILE_OUTPUT` with the same source/tools and another fresh output. Each command
requires Bubblewrap and a private transient user scope. It verifies4GiB/no-swap/
one-core limits before executing. Inputs mount read-only; HOME/network/PID/user/
mount namespaces are separate. Bare inner entry points refuse on the host.

The compiler is0.4.24 with optimizer200/Byzantium target. Chain and network IDs
are1337, hardfork Merge, time pinned, no fork, with persisted private LevelDB.
The public Bridge development mnemonic is used solely to reproduce deployment
addresses; it is not a wallet/storage recommendation. The two created galaxies
have no network keys and no ships are booted. Future ship keys must be fresh.

Nine contracts/13 receipts, bytecode, ownership, genesis and latest block survive
reopening. Separate wrong-chain/wrong-genesis nodes and absent contract code
refuse admission without sending transactions. These are fixture checks, not an
implemented runner network classification. JSON-RPC serving, fresh Urbit network
keys, the boot gateway, actual pinned-pill observation, messaging and ship restart
remain separate gates. See `docs/private-chain-research.md` and M216.

For the actual pinned Vere's embedded request-generation check, run the same
outer command with `--stage boot-requests --vere PINNED_BINARY`, a fresh output,
and no `--compiled`. This executes `vere eval` only, within the same admitted
boundary, and decodes point/DNS/256-galaxy requests. It does not boot a ship or
make an HTTP request. The binary digest is explicitly pinned in the driver.

`--stage key-validation` requires `--compiled`, `--deployment` and `--vere`.
It copies the stopped development chain, generates two fresh Suite B network keys
using the pinned ivory's `crub` interface, registers them and validates public
responses through the actual dawn parser/key validator. Wrong key/revision
responses must refuse. It does not start an HTTP server or ship. Keep the output
private: `secrets/` contains fresh disposable key files and generation output;
exclude that directory from reports, evidence copies and source control. Only
public keys/receipts and file hashes belong in development memories.

## Official-runtime native keyed peers

`--stage native-peers` requires `--compiled`, `--deployment`, `--vere` and
`--pill` in addition to the same source/tools inputs. Use a fresh private output:

```sh
python3 -B scripts/qualification/private-chains/run.py \
  --stage native-peers --sources SOURCE --tools TOOLS \
  --compiled COMPILE_OUTPUT --deployment STOPPED_DEPLOYMENT \
  --vere PINNED_VERE --pill PINNED_PILL \
  --output /tmp/omabit-private-chain-native-peers-NAME
```

This stage verifies the official binary and pill hashes, copies the stopped chain,
generates fresh Urbit network keys and records every helper hash. It admits a
6GiB/no-swap/one-core private scope only when at least8GiB is available, with a
900-second outer deadline. All existing foundation stages retain their4GiB and
180-second limits. The namespace exposes only private loopback and mounts no host
HOME, desktop bus, engine or keyring socket. No upstream cleanup script runs.

The test uses distinct authenticated sessions on two named keyed piers. It checks
peer-key observation, bidirectional messages, an intentional Gall rejection and
original-session receiver restart. A pinned disposable Gall agent then checks
four persistent writes and original/fresh subscriptions: queued state, saved event
cursor, reconciliation and explicit channel deletion. No uncertain write is retried.
The helper's host guard refuses before process or HTTP construction; run its narrow
checks with `python3 -B scripts/qualification/private-chains/test_native_entry.py`.

The pinned official runtime ignores SIGTERM immediately after keyed boot because
of a known signal-cleanup defect. This fixture deliberately uses acknowledged
`+hood/exit` with a hood Lens sink for each initial process and requires actual
exit0; the reopened receiver restores normal SIGTERM handling. A sent request
alone is never a successful Stop. This is a qualified native test lifecycle, not
an implemented runner/container shutdown contract. Docker integration remains open.

Keep the output private. Fresh key inputs and raw bootstrap logs remain in the
lab and must be excluded from public reports. Retained test piers themselves hold
networking keys, so their evidence belongs in private local storage. These keys
and the public development-chain mnemonic are not wallet custody mechanisms.
See M235–M237 and the private-chain research record for evidence and limitations.

## Keyed Docker lifecycle qualification

`docker_runtime.py` is a guest-only helper for the next no-NIC Docker experiment.
`test_docker_runtime.py` runs17 inert tests and never contacts an engine. Exact
container bindings and a dedicated rootless guest socket are required. Initial
keyed Stop uses an acknowledged Lens exit and observed exit0; reopen uses TERM.
Lost dispatch replies remain uncertain without resend. There is no forced Docker
stop fallback. This helper does not yet qualify Docker messaging or integrate
private chains into the runner. See M338 and the corresponding diagnostics.

The first Docker-chain VM (M339) found that Docker's containerd image store reports
the manifest digest. Admission accepts that verified digest or the config digest,
with the exact pinned reference required in either case. Tags still refuse. This
is a corrected source boundary; the first VM never started the gateway, so Docker
private-chain messaging and recovery remain pending a fresh run.

Private gateway diagnostics can explicitly capture both Docker log streams for one
exact container with a200-line tail and4MiB output bound. Retain these logs only in
the private lab; generic command failures still omit stderr and key material.

The guest-only `docker_case.py`, `docker_peer.py` and `docker_gateway.cjs` now retain
the exact sources of the active isolated Docker experiment. They are not host
commands and do not accept an existing Docker socket. The case checks the fresh
no-NIC guest before constructing an engine; peer HTTP and event entry points have
the same early refusal. Run the40 inert preparation/audit/lifecycle/entry checks from this directory:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B -m unittest test_docker_appliance test_docker_inspect test_docker_runtime test_docker_entry
```

The gateway pins its glibc native-addon selection and requires the complete tested
Node/addon library set. Its public256-galaxy preflight uses120s after a read-only VM
reproduced the30s timeout and completed in71.45s. Other evaluator calls retain30s;
failures retain only scalar status/timing metadata separately from private logs.
These source checks and gateway readiness do not certify Docker ship messaging or
recovery. The reusable VM preparation/launch interface and full stopped-disk audit
remain pending; do not invoke these inner entry points on the host. See M344.


M346 now supersedes the pending Docker peer/recovery status above. A fresh no-NIC
VM with the official image passes six acknowledgments/one intentional rejection,
four unreplayed persistent writes, seven scries, six snapshots, two channel
closures and same-pier receiver replacement with original sessions/cursor. All
three ship stops, the gateway and VM shut down cleanly; the independent read-only
audit preserves disk hashes. See `docs/diagnostics/2026-10-25-keyed-docker-recovery.md`.

M347 integrates the tested resource/backing settings into the common
`../imports/appliance.py boot` source entry point with11 inert regression checks.
A reusable private-chain preparation command and full execution through that
source entry point remain next. This qualification does not yet provide public
runner private-chain lifecycle/classification or a supported release artifact.

## Prepare and launch a private Docker appliance

`docker_appliance.py` prepares code and disk inputs only. Select the pristine
12GiB import appliance, the pinned foundation sources/tools and successful compile,
the original stopped **unkeyed** deployment, the complete Node/addon runtime bundle,
and the candidate runner and pinned official Vere. The foundation and runtime
preparation records above describe these inputs. No dependencies are fetched or
installed by this command; Go compilation is offline. It never invokes Docker,
boots a VM, reads a pier or creates network keys.

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/qualification/private-chains/docker_appliance.py \
  --base-appliance PRISTINE_APPLIANCE --tools PINNED_TOOLS \
  --sources PINNED_SOURCES --compiled PASSED_COMPILATION \
  --deployment STOPPED_UNKEYED_DEPLOYMENT --node-runtime VERIFIED_NODE_BUNDLE \
  --runner CANDIDATE_RUNNER --vere PINNED_VERE --output FRESH_PRIVATE_APPLIANCE
```

The output must be new and outside all input trees. Both its filesystem and
`/tmp` must retain8GiB headroom. A failed partial preparation is preserved; inspect
its failure and use a new output, without rerunning a completed build. A refusal
before output creation can be retried after its specific prerequisite is restored.
The command validates pins, foundation chain metadata, original/candidate runner
bindings and Node/addon libraries, seals a bounded regular-file payload, rechecks
its expanded inventory, and publishes `build.json` only after compilation succeeds.
The pristine base remains read-only; the new appliance contains an empty bind target.

Review that manifest, then use the common launcher with an explicit fresh run:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/qualification/imports/appliance.py boot \
  --appliance FRESH_PRIVATE_APPLIANCE --tools VERIFIED_QEMU_TOOLS --cpu-cores 2 \
  --output /tmp/omabit-import-guest-run-private-chain-NAME
```

The launcher admits16GiB available host memory, uses a12GiB/no-swap/128-task scope,
7GiB guest RAM, a3600s deadline and2GiB running storage reserve. Its VM has no NIC
and its own rootless Docker. Fresh keys, chain state, piers and private logs remain
inside its private writable overlay; export only the bounded audited metadata and
source/hashes. Neither preparation nor qualification provides public runner
private-chain project lifecycle support. The source-entry runtime trial is pending.

## Audit a completed private Docker run

After the original qualification has shut down and its exact scope is inactive:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/qualification/private-chains/docker_inspect.py \
  --run /tmp/omabit-import-guest-run-private-chain-NAME --tools VERIFIED_QEMU_TOOLS \
  --output /tmp/omabit-private-chain-inspection-NAME
```

The source command refuses active, incomplete, foreign or changed runs before
launch. Its separate no-NIC inspection VM has a2GiB/no-swap/one-core scope and
read-only disk bindings. The guest mounts ext4 read-only with journal replay
disabled; it starts no Docker daemon or ship. Both base and overlay hashes must
remain unchanged, and the inspection scope must stop cleanly before success.
The240s/2MiB console bound and256MiB temporary-storage admission remain explicit.

The guest reader checks the message, shutdown, persistent-state and subscription
receipts, exports fixed public metadata and hashes, and omits raw application
values and error bodies. Unknown fields are never copied wholesale. Write receipt
text is hashed. Portable synthetic tests replace the original machine-local test
fixture; the VM init refuses host execution before filesystem effects.
The new source-entry audit runtime is pending; M346 preserves the earlier qualified
reader and its exact successful output independently.

M352 now qualifies the repository preparation, common source launcher and new
source auditor end to end on a fresh no-NIC VM. Persistent state/subscription
recovery and clean shutdown all pass; the audit preserves both disk hashes. This
supersedes the pending source-entry/audit statements above. Public runner private
project lifecycle, host publication and simultaneous duplicate-project isolation
remain separate. See `docs/diagnostics/2026-10-25-private-docker-source-runtime.md`.

### Retained gateway candidate (M355; runtime pending)

`gateway_resume_contract.cjs` and `docker_gateway_resume.cjs` separate reopening a
stopped qualified chain from `docker_gateway.cjs` initialization. The existing
preparer still packages the original initializer. The resume candidate has no
chain-copy, network-key generation, transaction or mining path, and refuses host
execution before runtime dependency loading. Its pure checks run with:

```sh
node --test --test-reporter=tap scripts/qualification/private-chains/test_gateway_resume.cjs
```

Actual use requires a dedicated no-NIC VM controller with freshly verified prior
gateway/container/root/input bindings. That controller and actual Ganache reopen
are not yet qualified. Do not invoke this candidate on a host or substitute it into
an existing installed service. M355 covers source admission only, not public
private-chain lifecycle or arbitrary key/head generations.

M356 adds explicit `docker_appliance.py --gateway-resume` qualification preparation
and the ownership-bound `docker_resume_case.py` guest controller. M357 extends
`docker_inspect.py` to require a separately bound resume result for those builds.
The common reader still refuses host invocation and uses only a stopped disk in
a no-NIC read-only/noload guest.69 reader checks and50 Python tests pass. The first
fresh VM10 runtime and its disk audit are pending; these source gates do not enable
installed/private runner lifecycle.

M358 adds `--gateway-peer-recovery` together with `--gateway-resume`. This is an
explicit next fixture: separate root-owned client checkpoint, original sessions,
same-pier replacements in the new namespace, queued subscription recovery and no
replayed writes. Its public auditor never exports client cookies or raw values.
55 Python/109 Go checks pass. The first preparation was refused by quota before
output creation; no peer-resume VM has run. Preserve the8GiB preparation reserve.

## Read-only runtime input review

The development collector's `private-chain-inputs --help` describes inspection of
an already prepared sealed dependency generation. It accepts separate manifest and
foundation review hashes, verifies every file, and executes nothing. The same
library is available in `runtime-inputs/main.go` for source qualification. This is
not appliance preparation, chain initialization or runner lifecycle dispatch. See
`docs/private-chain-runtime-inputs.md`; current gateway inputs remain guest-only.


`prepare_runtime_inputs.py` extracts an inert dependency generation from a
separately reviewed private-chain appliance, verifying build/inventory/payload
and pill hashes. It starts no code or database library. See
[the preparation and independent inspection commands](../../../docs/private-chain-runtime-inputs.md).
Output must be fresh under a private canonical parent. Preserve failed outputs;
source-keyed piers, project keys and client sessions are not bundle inputs.
Run its eight inert boundary tests with
`PYTHONDONTWRITEBYTECODE=1 python3 -B -m unittest test_runtime_input_preparation`
from this directory. Passing preparation still requires the separate Go reader
and does not qualify supported runtime delivery or runner dispatch.

M367 now qualifies the complete `--gateway-resume --gateway-peer-recovery` path
through a fresh no-NIC VM and independent read-only audit. Exact piers, chain/head,
original sessions and queued subscription intent survive replacement without write
replay. Historical namespace inode values may repeat after the old object dies;
current membership and exact container/generation ownership remain required.
This does not implement public runner startup or qualify HTTP publication.

`--http-publication` additionally requires both recovery flags. It selects a
separate guest entry point and exact owned ordinary bridge, exposing only the
two Eyre ports on that no-NIC VM's IPv4 loopback. The direct retained-session
client uses no new login; RPC/admin/IPv6 refusal and direct state/subscription
recovery must pass a separately required read-only audit. Run
`python3 -B -m unittest test_docker_publication test_docker_published_client` from
this directory for the inert boundary tests. M368 records source qualification;
the actual publication VM is pending. This is not a public runner runtime command.

M372 qualifies `--http-publication` together with both gateway recovery flags.
VM14's original sessions and subscriptions recover through direct guest-loopback
HTTP; RPC/admin/IPv6 probes refuse. Independent audit29 passes on its unchanged
stopped disk. The reader requires four SSE acknowledgements in addition to
subscribe/two deletes (seven PUTs). Audits27/28 preserve the earlier counting
failure and diagnostic evidence. No public runner dispatch or simultaneous-project
claim follows from this single-project publication gate.

The `docker_appliance.py --project-isolation` case is mutually exclusive with
recovery/publication modes. It runs both fresh project pairs concurrently through
crossed-session, independent-state/subscription and ordered-shutdown comparisons.
The launcher requires32GiB available memory, a24GiB no-swap scope and a14GiB
no-NIC guest. `docker_inspect.py` derives its mandatory isolation audit from the
frozen build; absence of that proof cannot pass. M373 source checks/preparation
pass; actual runtime and stopped-disk auditing remain pending.

### First-start HTTP publication

`docker_appliance.py --initial-http-publication` selects publication from the first
gateway initialization. It is mutually exclusive with `--gateway-resume`,
`--gateway-peer-recovery`, `--http-publication` and `--project-isolation`. The
default case remains loopback-only. The single-pair16GiB admission/12GiB scope,
7GiB no-NIC guest and two-core limit are unchanged.

Only the two ordinary Eyre ports are published on the guest's IPv4 loopback.
The owned bridge's exact private subnet/gateway/routes and interface set must
match; the gateway's read-only RPC and both administrative listeners remain
unpublished. The no-NIC VM is essential to the experiment's external-network
isolation: an ordinary Docker bridge alone is not that boundary.

The original four-write receiver-restart transcript remains separate. A new
direct client uses the original sessions to verify two additional writes, four
subscription snapshots, three scries and closure of two new channels. It never
logs in again or resends an uncertain write. The mandatory independent reader
checks the initial head hash, session-file hashes, complete state, receipts,
cursor order, network/port bindings and direct request coverage. This initial
case uses eight channel PUTs; the replacement case continues to require seven.

Run the inert selection/refusal tests from this directory with:

```sh
PYTHONPATH=../fake-networks PYTHONDONTWRITEBYTECODE=1 python3 -B -m unittest test_initial_publication test_initial_publication_modes
node test_gateway_initial.cjs
```

The Node test extracts only the marked pure contract functions; it never evaluates
the initializer entry point.
M380 records the source pass. M381 records verified appliance20 and running VM15;
first-start runtime and its stopped-disk audit are still pending at that checkpoint.
Do not treat source checks or gateway readiness alone as a runtime pass.

This initial-only appliance intentionally lacks retained-gateway programs and
cannot be fed to `prepare_runtime_inputs.py`, which requires a separately reviewed
resume appliance with the complete initialization/resume dependency set. Existing
sealed generations keep their original hashes; a changed initializer needs a new
reviewed generation. Neither form is a published runner runtime or wallet store.

M384 supersedes the first-start runtime-pending statement above: fresh VM15 and
independent audit30 pass23 direct requests, two new writes, four SSE reads and
three scries using original sessions, with both channels closed and clean shutdown.
See [runtime evidence](../../../docs/diagnostics/2026-10-25-private-initial-publication-runtime.md).
Concurrent projects, concrete runner/public dispatch and supported deployment
remain separate qualification gates.
