# Isolated development and acceptance

Start by listing which effects the plugin can produce: credentials, files,
subscriptions/pokes, notifications, launches, theme/config changes, IPC, timers
and hooks. Label experimental installations separately and isolate every one of
those effects. A changed manifest ID alone is insufficient.

Use a separate worktree and experimental plugin ID/name. Give the instance private
HOME/XDG roots, profile-bound account metadata, keyring attributes, cache, lock files,
IPC names, launcher IDs and hook paths. Do not attach it to the production desktop
bus or compositor. Use a private desktop or an inert adapter until full isolation
has been tested. Read the plugin's actual helper environment: inherited paths or
desktop sockets can defeat apparent isolation.

An Omabit desktop profile isolates collector/desktop state, not remote ships or
host Docker/systemd services. Runner operations still require the intended runner,
instance and current revision. Runtime/network/installer tests use a disposable
engine/VM and new fake identities. Do not inspect protected sources to prove they
were excluded. Use Omabit's operator skill for management commands rather than
inventing a direct Docker workaround.

Collections should make stable/dev/edge switching explicit and reversible, with
the actual loaded artifacts, endpoints and state visible. This is a roadmap
requirement; current per-plugin profiles are not evidence that collection switching
already works. Do not invent collection commands. Before claiming support, test
pending work, logout, crashes, activation failure and rollback across the entire
set, including credentials, IPC, hooks and launchers. Old desktop artifacts must
not silently downgrade runner/ship state.

Do not implement a collection switch as one enable/disable list update. The
inspected native loader starts incoming services before destroying outgoing ones;
`keepLoaded` can retain old code under new manifest metadata. Quiesce and disable
the old collection, confirm its owned watchers/helpers have stopped, then enable
immutable candidate paths and acknowledge actual loaded generations. Preserve
uncertain work; rollback follows the same drain boundary. These behaviors are
characterized with copied native loader methods and synthetic services, not yet
qualified for the full real-plugin collection. See the source repository's
`docs/collection-activation.md` and its linked evidence.

Use isolated project fake networks for the multiplayer happy path when that
capability is implemented and qualified. A filesystem project label or duplicate
fake ship name does not establish networking. Private Ethereum development uses
a local chain with pinned chain ID/contracts, RPC, disposable identities, pill
and Vere configuration. Public testnets are outside the initial scope. Do not
substitute an alternate Ethereum URL for a verified private-chain setup.

The source private-chain harness now demonstrates official keyed Docker peer state
and subscription recovery. Use its README for the separate preparation, no-NIC VM
and stopped-disk audit commands within the authorized test boundary. The collector's
`private-chain-plan` reviews public chain/genesis/contracts metadata offline; it
does not create a chain and cannot enter fake-project registration. The separate
`private-chain-register`/`private-chain-projects` commands reserve and inspect public
plans on capable pinned runners; they do not create keys or launch runtimes. Preserve the
private project/generation binding even when another project repeats the same
ship names or genesis. Local `private-chain-inputs` can hash a prepared sealed dependency
generation against separate review hashes without executing it. Its result grants
no initialization or startup authority; gateway inputs remain guest-only candidates. These disposable network keys and public development seeds
are not a wallet custody design.

Acceptance should exercise behavior across the boundary that matters:

- Sign in without Omabit; approve through Omabit when available; both reach the
  same direct client path. Separate plugins and profiles retain separate sessions.
- Read, subscribe and write against a disposable backend; recover subscription
  state after reconnect without flooding it or duplicating uncertain writes.
- Expire or revoke one session; preserve other applications and newer standalone
  sign-ins. Preserve unresolved attempts rather than silently starting over.
- Start without the backend; install the exact reviewed source, handle permits,
  reach readiness and resume after interrupted observation. Exercise incompatibility
  and both desktop/backend update orders.
- Render the actual UI states and use real helpers. Offscreen components with
  inert adapters are useful but do not qualify live desktop hooks or launchers.
- Switch a stable/dev/edge collection; verify endpoints, credentials and every
  side effect stay in their selected profile, including after failed activation.

For multiplayer writes, verify the receiving application's acknowledgement and
observable state, not just HTTP submission. Exercise the exact wire/mark conversion:
a generic `%noun` poke can arrive with vase type `*` across Ames. Validate its noun
shape before interpreting it; do not assume the caller's @t aura survives. A typed
application mark should describe the intended production protocol.

For restart recovery, write unique values before stopping the disposable receiver,
read the persisted state after it resumes, then verify queued/live updates and
fresh-subscription reconciliation. Keep the original session when testing session
survival. A clean ship restart is distinct from a Gall agent upgrade/on-load, a
TCP reconnect, an expired channel and a power failure; report each tested boundary
separately. Reconcile an uncertain write instead of silently submitting it again.

Record the source/artifact identity, boundary, actual behavior, retained failure
evidence and remaining gaps. Use narrow fixture checks before live disposable tests.
Do not expand a successful helper test into a claim about full desktop/runtime
behavior. Never use the user's installed plugins or valuable piers as fixtures.

Wallet/key storage remains research, not an incidental feature. Record concrete
findings about same-user plugin authority, credential-store access and desktop
isolation in the platform security notes. Do not place signing keys into the Eyre
connection format or claim that private files/keyring labels protect them from
arbitrary trusted-by-the-user desktop plugin code.
