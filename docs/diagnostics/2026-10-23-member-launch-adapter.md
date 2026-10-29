# Initial member adapter and queue — 2026-10-23

The concrete private adapter reopens the original project engine/image/metadata
contract, reconstructs exact peer launch contracts, and accepts only a fresh target
workspace and observed empty parent. It requires a separately staged pinned pill;
it copies and verifies that artifact in a new private workspace after committed
dispatch. Known directory identity survives subsequent failure. The published
copy must remain exclusive/read-only and no unexpected workspace files are accepted.
Dependency transfers stay in the separate visible installation/staging flow.

Before Create and Start, the driver requires fresh complete namespace observation,
verified parent identities, workspace/pill checks and the negative mount-conflict
preflight. It creates only the fixed member contract, retains an exact ID even with
a late client error, and verifies the Start acknowledgement. A partial pier before
Start refuses boot without deleting it. Running observation permits the new named
pier child, while making no claim that the application or pier is initialized.
No stopped/recovery container is silently accepted and no uncertain effect replays.

The private server queue shares the normal work gate and update-worker fence.
Setup is revalidated after acquiring that gate; cancellation, setup failure or
retained maintenance leave an interrupted unbound launch receipt. No public route,
startup scheduling or capability advertisement has been added.

Eleven injected-engine/temporary-filesystem scenarios pass, including late/missing
replies, partial storage, namespace differences and unavailable/changed artifacts.
Queue tests cover shared gates, update exclusion, replay and setup failure. Focused
project race suites and vet pass. Schema27 is unchanged; no real engine, ship or
installed plugin was accessed. A guarded fresh-VM case is the next runtime gate.
