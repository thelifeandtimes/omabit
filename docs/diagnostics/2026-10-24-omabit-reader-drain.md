# Omabit monitor and setup reader shutdown

The service now registers readers outside its own object tree. Runtime logs,
project history and SSH suggestions retain pending exit state until the actual
Quickshell Process exit signal. Shutdown closes admission first and waits for these
readers as well as its existing service-owned helpers. Removing an object with an
unconfirmed exit latches reader-owner-lost: object disappearance cannot authorize
replacement. A process that fails to start does not leave a false pending exit.
This is cooperative accounting, not an OS sandbox or a full collection receipt.

The optional --readers mode of scripts/test-omabit-service-drain-offscreen.py
loads actual Service, MonitorWindow, RuntimePane, RemoteRunnerPane/SSHHostField and
ProjectNetworksPane/ProjectNetworks components. The separate setup controls use
the same service but live outside its object tree. Exact-argv helpers, Commons
tokens and window focus are inert; Quickshell, Process, timers and IPC are real.
The fixture has private HOME/XDG/mount/PID/network namespaces, no NIC route and no
host desktop, keyring, runner, engine or SSH sockets.

The initial trial exposed a real stale-result bug: SSH suggestions were accepted
after drain. That failure is retained; the corrected callback now rejects results
after admission closes or the field becomes hidden. It also corrects the fixture's
missing instance config that initially caused monitor binding warnings.

Draft and applied runs pass stable→dev→stable→edge→stable. All three auxiliary
readers delay SIGTERM completion; feed exit alone never yields closure. Late
results are ignored, refreshes/toggles cannot restart readers, and exact old child
identities have exited before each replacement. Foreground work refuses drain;
late service reads still prevent closure. Destroying live reader controls retains
reader-owner-lost even after the fixture independently verifies the exact child
exits. Failed executable start does not wedge shutdown. The original service-only
baseline and existing project UI regression also pass. Applied source hashes,
host-entry refusal, clean shell exits and inactive scopes are independently checked.

M266 supersedes M256's monitor-stub-only lifecycle evidence for these readers and
narrows M265's monitor shutdown gap. Real collector/auth/clipboard descendants,
native compositor and general durable collection transitions remain separate.
collectionReady remains false. No protected ship, installed plugin, production
service, credential or desktop configuration was touched.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-omabit-reader-drain`.
