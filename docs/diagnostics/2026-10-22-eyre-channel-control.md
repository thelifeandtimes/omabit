# Pinned runtime direct Eyre control

2026-10-22. **The same-ship Eyre channel control passes.** The exact static Vere
binary extracted from the verified pinned image (`4.6-5a7614f`, SHA-256
`b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0`) and the pinned
pill booted one new fake in a private Bubblewrap namespace. A checked user scope
capped it at 4 GiB, no swap, one CPU and 128 tasks. No Docker, existing pier,
host network, credentials, home or desktop sockets were available.
[Machine-readable evidence](2026-10-22-eyre-channel-control.json).

The probe signed in directly, read desk inventory, and sent a fresh nonce to
`%hood` using the standard Eyre channel `%helm-hi` poke. It matched the Gall
application acknowledgement by channel, request ID, response kind and SSE event ID.
A second private channel sent `fail` and received the expected `poke-hi-fail`
application rejection. Both channels were acknowledged and deleted. The fake
stopped with exit 0; the exact scope is inactive. Original artifacts, private
logs and pier remain at `/tmp/omabit-thread-probe-20261022-07`.

## What the preceding diagnostics established

The two Docker trials remain failed transport gates. Native trials used the exact
image binary to separate their HTTP problem from Docker and CPU emulation:

- Trial01 stopped before ship creation on a Python name collision; it was fixed.
- Trials02/03 exposed malformed Hoon hex formatting in the codec diagnostic.
- Trials04/05 verified the noun encoding against the live runtime, then reproduced
  the generic `%noun`-input Spider `hi` timeout. The pill's actual `hi` and Spider
  sources were retrieved read-only. A typed-vase mismatch is a source inference;
  no root-cause fix or upstream defect is claimed.
- Trial06 received the exact positive channel acknowledgement, but failed an
  incorrect receiver-stdout assertion. `%helm-hi` emits Dill terminal output;
  these batch runtimes do not provide that transcript.
- Trial07 passes positive and deliberate negative **application** receipts.
  It does not claim a cross-ship, Docker or restart pass.

All seven exact scopes are inactive. Every native test ship stopped cleanly;
trial01 created none. Failed artifacts were preserved, not relabeled as successes.

## Revised network checkpoint

The Docker harness now uses this channel protocol for its same-ship controls,
bidirectional messages, remote rejection and messages after restart. Each poke
has a fresh channel and request binding; accepted submission is insufficient.
The bounded SSE observation must contain one complete matching application receipt.
A deliberate rejection must contain `poke-hi-fail`, rather than a generic transport
or mark-conversion error. No uncertain poke is retried. Cookie jars stay private.

The observation intentionally ends its open stream after 60 seconds. Curl exit28
is acceptable only for that bounded stream, with HTTP200 and the exact receipt;
it never turns a missing acknowledgement into success. The channel is then
acknowledged and closed. The CLI does not proxy application data.

Three offline tests cover accepted/rejected outcomes, wrong request IDs, boolean
IDs, duplicate receipts, truncation, oversized data and unrelated errors. They pass,
as do Python parsing and diff checks. The revised Docker case has **not yet run**.
Cross-project duplicate identities, persistent application state/subscriptions,
runner integration and endpoint publication remain later gates.
