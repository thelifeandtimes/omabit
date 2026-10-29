# Persistent application recovery through the runner: source fixture

The next separately opt-in no-NIC guest case uses the real private runner lifecycle
and the existing disposable Gall state fixture. It targets ~zod, initially writes
from ~nec, and retains the same zod session/channel/cursor over two replacements.
The peer stays stopped afterward; the remaining writes originate locally on zod.
This does not claim both members can resume after an unrelated peer replacement.

Four separate helper invocations persist phase intent before effects and refuse
replay, skipped phases or uncertain writes. Only acknowledged unique entries and
the original cursor/channel are restored. Each replacement checks the queued update,
existing subscription, further writes and fresh-client reconciliation. Two cycle
snapshots remain alongside cumulative receipts, with eight writes total.

The helper checks QEMU/boot/hostname/owner/no-NIC markers before file or engine
access. It binds the exact private engine, anchor labels/namespace and fresh pier
identities. Cookies stay only in the runner anchor's existing1MiB /urbit tmpfs;
the readonly /tmp path from the earlier lower-level fixture is unsuitable here.
The actual prior runner receipt confirms this mount and the /srv/urbit/piers root.
No session secret is placed in argv or the evidence. Later phases never fetch
+code or authenticate again. The fixture's dedicated labels, ports and paths are
independent of all user ships and installed plugins.

Eleven phase/transport tests and eight routing/existing state tests pass. Five VM
entry points skip ordinarily; explicit Go/Python host attempts refuse before runtime
access, and the app flag requires the private member fixture. Compilation/vet pass.
The first compilation's optional-pointer mismatch remains in the evidence log.
The full schema35 managed suite in M113 predates these fixture-only additions.

This extends M056's lower-level state recovery into a runner-connected test design
and source fixture. It does not supersede any pending runtime gate. Running VM22
was frozen before this wiring; a new appliance must embed and verify this case.
