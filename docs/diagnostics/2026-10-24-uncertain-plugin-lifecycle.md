# An idle UI is not proof that a DM write is resolved

The actual copied DMs overlay/bridge submits exactly one synthetic chat poke to the
private Eyre fixture. Its HTTP acknowledgement is returned but the Gall receipt
is withheld. While waiting, the overlay is busy and retains its request ID/text.
After its normal acknowledgement deadline, it reports an unknown send outcome,
clears busy and pendingSend, and still retains the text. No resend occurs; both
original helpers and subscription streams remain active during observation.

This passes the characterization test, not a durable transition gate. A collection
coordinator cannot treat `busy=false` as evidence that all writes are resolved.
It needs explicit unresolved-operation state and a durable handoff/reconciliation
contract before unloading an outgoing plugin. The held receipt is a deliberately
synthetic condition, not a claim about a real ship's application state.

The final private shell teardown is clean and all observed streams close. Actual
plugin helper hashes are unchanged; only fixture QML inspection/IPC and the
synthetic server differ. The experimental fixture remains unapplied. The exact
scope is inactive and retained evidence excludes the private keyring database.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-uncertain-plugin-lifecycle`.
