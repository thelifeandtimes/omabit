# Published duplicate-project isolation — 2026-10-23

Fresh no-NIC VM23-03 passed the combined transport/state/isolation gate in
2427.62seconds. Each independent zod/nec pair completed57 direct published HTTP
requests, five writes and17 state observations. Same-pier receiver replacement
preserved original sessions and queued state; the existing subscription resumed,
a new subscription reconciled, and both channels were deleted cleanly. Both
pairs completed their three expected clean ship stops; anchors exited143 without
OOM, and the guest powered off normally.

The projects use different bridges/namespaces and four distinct IPv4 loopback
publications. Four same-identity cross-project session attempts returned401.
Independent fifth writes preserved the other project's state with no entry
overlap. IPv6 and bridge-reachable administrative endpoints were refused.

The original host driver exited1 because journald split the56KiB final JSON
across two same-process records; its `guest_result` remains null in the original
report. This was a reporting failure, not a VM or application failure. A bounded
same-process continuation parser now passes three source tests. Independent
verification reconstructed the exact original console value and required full
equality with the guest disk's result before checking every assertion. No original
receipt was rewritten. All17 extracted receipt files, original artifacts/disks and
37 retained metadata files were hash-verified (the manifest is authoritative).

The application receipt SHA256 is
`8d8584298cc83d64dd09202dfe8fcd84b64d63242cf2f4371e750d7f37a978a4`.
Original images remain in task evidence, with compatibility links at their old
temporary paths. Only the verified regenerable raw conversion was removed.
This closes M039's combined gate. Actual-ship anchor recovery and integration with
the new journaled runner lifecycle remain separate. The VM used an older frozen
harness, so this does not qualify M048's adapter. Production remains untouched.
