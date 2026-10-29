# Package the native-addon library dependencies

Run06 passes fresh package verification and owner/engine installation, then its
gateway fails before keys or peers. The glibc addon is selected correctly, but
libpthread.so.0 is missing. The original package builder collected libraries linked
by Node alone; addons loaded later can require additional compatibility libraries.
The previous probe had access to host /usr libraries, so it established native
addon selection/loading but did not prove the supplied runtime was self-contained.
This supersedes that completeness assumption in M342, not its observed results.

Dependency inspection covers14 pinned Linux x64 non-musl addons. Only the14,360-byte
libpthread.so.0 is missing from the existing Node library set. A new probe runs the
same explicit dynamic loader and library path with no host /usr, /lib, /lib64 or
loader cache. The original set reproduces the missing-library error. Adding the
pinned dependency makes Ganache load and LevelDB preserve a value through close
and reopen. Test data stays in its private lab; only result booleans and code
hashes are retained. The next appliance packages exactly this validated set and
its manifest, checking every addon/library hash first.

The independent auditor also now hashes private error text rather than exporting
it: a rejected Hoon expression could contain a fresh network key. Fixed error
codes and /gateway.cjs source locations remain available. Three top-level tests
plus four negative cases pass, including an explicit private-value redaction test.
Only pre-key startup logs can be exposed, after exact source-hash and absent-key/
peer checks. This run's read-only audit leaves both disk hashes unchanged. Its
387.50s VM shutdown is clean, but the gateway exit is1; no messaging pass is claimed.
The diagnosed no-ship machine disk is retired with source/audits/hashes retained.
Private runtime resource and isolation limits remain unchanged. No production
ships, tmux ships, temp-ur data or installed plugins were operated on.
