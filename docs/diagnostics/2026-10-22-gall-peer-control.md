# Pinned native Gall peer control

**Pass:** two newly created native fakes exchanged nonce-bearing Gall pokes in
both directions. A deliberate remote `fail` poke returned `poke-hi-fail`. After
cleanly stopping and resuming `~nec` from its existing pier, both directions passed
again using the original authenticated sessions. Both peers stopped cleanly; the
exact private scope is inactive. [Verified evidence](2026-10-22-gall-peer-control.json).

The pinned static Vere binary and pill ran in a private Bubblewrap namespace with
only loopback, no host home/desktop/Docker sockets, and a checked 8 GiB/no-swap/
one-core/128-task scope. Every artifact hash was verified. Separate cookie jars
worked for their own ship and were refused by the other ship.

The corrected probe uses the built-in local `khan-eval` thread with typed `ted-eval`
input. Its fixed program sends an ordinary Gall `%helm-hi` poke, waits for the
application acknowledgement, then returns a unique expected receipt. Six positive
receipts and the intentional negative receipt matched. The negative trace must
name the actual Gall `poke-hi-fail`; a generic HTTP failure cannot pass.

The preceding 4 GiB two-peer control reached its cap and failed with `oom-kill`.
It is retained as failed evidence, with no clean-shutdown claim. The fresh 8 GiB
run passed; no failed pier was reused. The single-ship positive/negative control
also passed separately. This establishes the protocol and native restart only.
The revised Docker transport/recovery case still requires its own successful run.

Six offline tests pass, covering response binding/truncation, failure distinctions,
and rejection of uncontrolled strings in the generated probe. No production ship,
plugin installation or existing Docker engine was accessed.
