# Pending application state survives supervised shell loss

The optional collection-recovery-state fixture extends M280 with two actual helper
writes immediately before shell SIGKILL. The synthetic peer accepts one DMs send
and one Theme settings write but withholds both acknowledgements. The fixture
captures the loaded activation, DMs journal byte digest and pending ID, and Theme's
uncertain intent ID, origin/ship/session fingerprint and desired-value digest.

Trial02 passes: the same cohort survives shell loss and original guardian cleanup,
with exactly two accepted writes and no replay. All17 channels close, four original
guardians certify their owned descendants exited, and six coordinator crash cases
still pass. The old activation binds its original guardian; the new empty shell's
guardian cannot replace it. Source hashes and closure receipts verify independently.
Both scopes are inactive. Trial01 is retained: the fixture incorrectly expected a
true return from the existing undefined-return DMs send IPC and an idle pendingSend
value of0 instead of-1. Correcting these assertions changed no plugin behavior.

M280 proved process/transport closure but did not contain unresolved application
writes. M281 adds that specific missing evidence; it does not reinterpret M280.
The format1 collection journal still retains reader-owner-lost at drain-previous.
Next add an explicit versioned prepared/verified recovery transaction, bound to
this old activation, original guardian and retained plugin state. Preserve pending
application work and credentials; process closure must never imply application ACK.

Real plugin components/helpers run under private identity instrumentation with
synthetic Eyre, inert Omabit/desktop controls and isolated HOME/XDG/PID/network.
This is not native loader qualification, a production recovery command, or proof
that arbitrary external helpers are absent. Guardian loss remains unconfirmed.
No production ships, tmux/temp-ur, installed plugins or credentials were accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-recovery-cohort`.
