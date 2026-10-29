# Private local Ethereum chain — source constraints

Initial research, 2026-10-22; source-only statements below describe that snapshot.
M216 (2026-10-24) advances the chain foundation: the guarded
`scripts/qualification/private-chains/` fixture now compiles/deploys the pinned
contracts and verifies admission and database reopening in a private namespace.
That foundation-only checkpoint did not boot ships; later native gates are recorded below. Wallet integration remains outside this work. See [evidence](diagnostics/2026-10-24-private-chain-foundation.md). The first supported path remains a private local development chain;
public testnets and mainnet forks are outside this initial goal.

The pinned Vere commit `5a7614f69bbc783736b08965fdc0a2534bb42e82` exposes
`--ethereum` separately from `--local` and `--fake`. A gateway URL alone is not
evidence that the ship uses the intended Azimuth network.

The local Arvo reference checkout at `50c7f5239610fc72263df7aa99b4d171cfe9d408`
has an explicit `%local` selection in `gen/azimuth/watch.hoon` and
`lib/dice.hoon`. Its `base-dev/lib/azimuth.hoon` defines a fixed local contract
set and chain ID 1337, with a comment tying the addresses to Bridge deployment.
Consequently, an arbitrary fresh chain/contract address is not automatically
compatible with the existing `%local` path. This is a source inference, and must
be checked against the actual pinned pill before implementation.

| Reference input | SHA256 |
| --- | --- |
| pkg/base-dev/lib/azimuth.hoon | f64ce94c0dbb8d7b5f28e0abb9e9d7d25e6169c8aabf59e7183cf84f9fe8ec45 |
| pkg/arvo/lib/dice.hoon | 3b6e3bfcf00dda7f1899b71d50a248e9af16fa208423a111689a8b58f9a5a7db |
| pkg/arvo/gen/azimuth/watch.hoon | 88c96b69a4a58e8558d4e79da48735ad74ad897785daeaa8ef58a1cf08409501 |

Bridge documents a local Ganache/Azimuth development flow and a fake-ship L2
setup. Treat it as reference material to qualify, not a script to invoke on the
user's desktop. [Bridge development documentation](https://github.com/urbit/bridge/blob/master/DEVELOPMENT.md).

Its inspected package scripts set chain ID 1337, bind the development node to
all interfaces, and use process-name cleanup. An Omabit lab must instead bind
inside its private boundary and terminate only its own child process/container.
Record the exact deployment/runtime inputs and preserve the user-visible project
identity. [Bridge package scripts](https://github.com/urbit/bridge/blob/master/package.json).

Azimuth's repository separates on-chain storage from Ecliptic's mutation interface
and documents local contract tests. Its mainnet-fork scripts are a different
workflow and are not a prerequisite here. [Azimuth source and test instructions](https://github.com/urbit/azimuth).

The next chain experiment should freeze the dependency/deployment artifacts, then
verify chain ID, genesis identity, contract addresses/code and fresh dev key
ownership in an isolated node. Qualify actual ship observation of those keys before
calling the setup a ship devnet. After that, test peer messages, restart and chain
state retention with exact project ownership. Never derive production identities
from public development seeds or present those seeds as wallet storage. The wallet
security research remains separate in platform-security-notes.md.

A collection profile should select a pinned chain/project artifact and explicit
ship endpoints; switching profiles must not silently reset the chain or repoint
existing ships. Preserve old project state while changing client collections.

## Keyed boot gateway constraint (M216)

The inspected Vere dawn implementation calls `point:give:dawn` and
`czar:give:dawn`; the Arvo runtime library serializes `getPoint` and `getDns`.
Its default is the roller's Azimuth endpoint. The `--ethereum` URL therefore
cannot be assumed to accept a raw Ethereum JSON-RPC server for keyed boot.
This is source evidence, not yet a captured request from the pinned binary.
The next bounded test should capture a freshly generated disposable key's boot
requests in a no-network lab, then qualify a private chain-backed gateway without
public fallback. Ship-key freshness and successful key observation remain gates.

M217 now confirms those generated methods using the exact pinned Vere binary's
embedded ivory (`vere eval`). This supersedes the inference-only qualifier for
request generation; it does not establish HTTP transport or boot success.
[Binary evidence](diagnostics/2026-10-24-private-boot-requests.md).

M218 adds chain-backed response and fresh-key interoperability through the pinned
ivory evaluator: point/galaxy/DNS parsing and key/revision acceptance/refusal pass.
The binary's Suite B generator is `crub`, not the current source's `cric` interface.
This advances the gateway adapter proof but does not qualify HTTP or ship boot.
Keep fresh key files private and excluded from reports. Private boot must replace
the development contracts' default public DNS with explicit project resolution.
[Key/response evidence](diagnostics/2026-10-24-private-key-validation.md).

M219–M223 advance HTTP/keyed bootstrap but leave clean shutdown unqualified.
Pinned Vere keyfile parsing restores signals twice and leaves SIGTERM ignored;
a same-launcher fake control passes. Private Lens exit also fails. See
[keyed signal diagnosis](diagnostics/2026-10-24-keyed-signal-diagnosis.md) and
[failed Lens trials](diagnostics/2026-10-24-private-lens-shutdown.md). Earlier
no-HTTP/no-ship statements are historical. A separately labeled runtime-source
experiment is in progress; no production binary or image is changed.

M224 qualifies fresh keyed SIGTERM stop and same-pier reopening with a separately
built experimental correction, including a failing-original/passing-candidate
signal regression. The official runtime remains unchanged and unqualified for
this fresh-keyed shutdown gate. Chain watcher, peer messages and Docker devnet
remain. [Evidence](diagnostics/2026-10-24-private-fixed-keyed.md).

M229 qualifies actual local key revision observation and Jael persistence across
keyed restart with the separately labeled experimental runtime. Lens generators
use `+` with an explicit app sink; watchdog interval must be configured before
creation, and the chain must advance past the30-block confirmation lag. An
identical-key configureKeys receipt is successful but does not rotate keys.
Independent stopped-chain audit verifies the actual changed keys and receipts.
Two peers and Docker remain pending. [Evidence](diagnostics/2026-10-24-private-watcher-recovery.md).

M233 qualifies two fresh keyed peers, authenticated bidirectional messages and
receiver restart using the experimental runtime. The copied chain and four final
databases pass independent audit. [Evidence](diagnostics/2026-10-24-private-peers-recovery.md).

M235 qualifies the official runtime's explicit Lens shutdown workaround: correct
`+hood/exit` source and hood sink cleanly stop the first keyed process; reopening
that pier restores normal SIGTERM behavior. This supersedes M223's invalid-source
Lens conclusion, not the independently established signal defect. A runner must
qualify and expose this lifecycle deliberately; never infer a successful Stop
from dispatching an unacknowledged request. Official peers/Docker remain separate.
[Evidence](diagnostics/2026-10-24-official-lens-recovery.md).

M236 repeats the two-peer transport/restart gate with the unchanged official binary:
initial keyed processes use acknowledged Lens exit, and reopened piers use SIGTERM.
Independent chain/public-key/receipt and final-pier audits pass. Native keyed
application-state/subscription recovery is next, followed by Docker/runner lifecycle
qualification. [Evidence](diagnostics/2026-10-24-official-private-peers.md).

M237 adds persistent Gall application state and original/fresh subscription recovery
to official keyed peers: four unique writes, seven scries, six snapshots/two channels
and retained session/cursor across clean restart. The copied-chain audit passes.
Docker/runner integration remains open.
[Evidence](diagnostics/2026-10-24-keyed-application-recovery.md).

M239 qualifies the integrated `private-chains/run.py --stage native-peers`
command from fresh inputs with full application/reconnect evidence and independent
source, stopped-pier and copied-chain audits. The repository now reproduces this
native happy path using the official runtime and explicit first Lens shutdown.
Docker/runner lifecycle support remains a separate implementation gate.
[Evidence](diagnostics/2026-10-24-native-keyed-recovery-runtime.md).

M339/M340 advance Docker fixture admission, not ship execution: recognize the exact
pinned config/manifest image identity, and explicitly override the Vere image's
`/urbit` volume when using it as a gateway. Both failed no-NIC VM runs refused
before gateway Start. Read-only disk audits preserve their evidence. The corrected
gateway uses an empty read-only tmpfs and explicit cwd; the fresh runtime gate is
pending. See the image-pin and gateway-mount diagnostics for exact scope.

M341–M344 advance the private Docker fixture: streamed package authentication,
explicit glibc addon selection and the full Node/addon library set now pass actual
gateway startup. The next gateway uses a measured120s allowance only for its public
256-galaxy preflight; other evaluator calls stay30s. Keyed Docker messaging remains
pending. See [evaluator diagnosis](diagnostics/2026-10-25-keyed-docker-evaluator-budget.md).

M346 now qualifies two official keyed Docker peers with persistent state, original/fresh subscriptions, same-pier restart and acknowledged clean shutdowns in a no-NIC VM. This supersedes the pending Docker peer gate above; runner integration and a reusable outer launcher remain. See [Docker recovery evidence](diagnostics/2026-10-25-keyed-docker-recovery.md).

M352 qualifies the complete repository preparation/launch/read-only-audit sequence
with fresh official Docker peers. This supersedes the pending reusable outer
launcher/auditor status above, while preserving M346 as independent prior evidence.
The offline private-chain planner (M349) cannot register or launch a project;
persistent runner integration, publication and project isolation remain open.

M355 separates retained gateway startup from the M346/M352 initializer. The latter
copies a chain and creates keys and therefore must not serve as a restart path.
The separate source candidate passes36 inert checks and host refusal; actual
Ganache reopen and ownership-aware gateway replacement remain pending. Its first
profile is limited to the qualified head0x50 and life1 keys, with no rotation or
initialization fallback. See [resume source](diagnostics/2026-10-25-private-gateway-resume-source.md).

M358 identifies gateway tmpfs cookies as a fixture-specific persistence gap for
whole-gateway recovery. The next source case retains client sessions/subscription
intent separately from project keys/piers and immutable runtime inputs.55 Python
and109 Go checks pass; actual recovery is pending. Preparation refused insufficient
quota before creating anything; no admission reserve was relaxed. See
[gateway peer source](diagnostics/2026-10-25-private-gateway-peer-source.md).

M360 replaces VM10's pending state with an overall pre-creation refusal: the chain
copy inherited0755, while the retained gate requires0700. Both directories were
canonical and correctly owned. Its original peer recovery independently passes.
The next initializer fixes only fresh-copy mode and refuses existing destinations;
the existing disk stays unchanged and privately retained. Packaging is not proof
of private directory modes. Corrected retained startup remains unqualified.
