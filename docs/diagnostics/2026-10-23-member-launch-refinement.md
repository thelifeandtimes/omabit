# Initial launch refinements after trial12 — 2026-10-23

Full pinned-pill hashing still occurs during workspace preparation and preflight,
before runtime observation. Each hash is bracketed by the same private workspace,
regular 0400 file, exact size, exclusive link count, owner, device/inode, birth,
modification and change timestamps. Create/Start require the identical verified
snapshot instead of hashing 217 MB again inside the runtime freshness window.
Atime is excluded because reading the file may change it. The snapshot is local
to one operation; reopening requires fresh hash verification. The 30-second
namespace and mount-preflight freshness bounds remain unchanged.

Thirteen artifact tests pass, including restored mtime after content tampering,
same-content inode replacement, permission changes, links, directory replacement,
cancellation, bad hashes and modification between hash and snapshot checks. The
concrete adapter also refuses mutations after observation and before Create/Start,
retaining exact dispatch phases without effects. These are local dispatch guards,
not sandboxing for hostile same-user software.

The member Docker contract now requires `--init` and verifies Init=true, matching
the earlier passing fake-network fixtures. The VM launches zod before nec and
associates HTTP checks with instance IDs, independently of sorted plan display.
Command changes alter the contract digest; no old recorded container is adopted.
Existing immutable launch receipts remain unchanged. The trial12 shutdown cause
is not claimed proven by these source changes.

Focused race regressions and vet pass. Appliance13's 100 source files, artifact
hashes and embedded test were independently verified before a separate fresh run.
Original trial12 disks and failed receipts remain retained. This record does not
claim a passing runtime, durable readiness, Stop/resume/recovery or public API.
