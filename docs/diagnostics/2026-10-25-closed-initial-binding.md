# Recovering adoption before binding publication

An initial activation can be canonically adopted before its authority runtime
binding is saved. If its original shell then dies, ordinary verified recovery
can close its channels but previously could not complete authority handoff.
Authority.bind_closed_initial now reconciles that historical binding only after
canonical original-owner cleanup verification, before unload/handoff. It checks
the original runtime observation hash, adopted instances, preconstruction guardian,
actual original ECHILD closure, closed gate and dead original guard/helper cohort.
It preserves the original reservation and journal. It never grants a new admission
or claims that an ended guard remains live. Exact publication-lost retries preserve
bytes. Existing reservation2/3 shapes and reader compatibility are unchanged.

Eight binding tests cover missing cleanup, changed runtime/closure/claim owner,
too-late reconciliation, exact retry, original-reader compatibility and subsequent
ordinary handoff. Four additional tests force atomic-publication/read interleaving.
Both legacy/shared readers previously rejected a valid replacement when the opened
old inode reached zero links before fstat. They now reopen the current named record
with eight bounded attempts, preserving rejection of missing, linked or malformed
evidence. No lock creation, state repair or format rewrite occurs on read.

The first 213-test run failed one existing guard-startup setup.100 instrumented
startup repetitions did not reproduce it. Controlled interleaving independently
reproduced the snapshot race in both readers; this is a demonstrated defect, but
cannot conclusively explain the uninstrumented first failure. A diagnostic driver
quoting error is also retained. The final217-test suite passes, including all prior
startup/rollback and historical-format cases.

The actual private native post-adoption test passes before and after the reader
fix. It kills the controller after adoption, loses the original shell, refuses
channel deletion and rejects a replacement guardian. Cleanup, recovered binding,
handoff, candidate enable and unload survive seven controller kills in total.
Stable/dev/stable constructs nine primary instances, establishes15 subscriptions,
closes ten channels, preserves six sessions and the exact unresolved DM, and issues
no new login or duplicate retry HTTP effects. Independent replay verifies the
canonical history, original closure/runtime identities and reconstructed exact
binding-publication hash. Reservation remains2. The prior pre-adoption regression
also passes five controller kills, ten subscriptions and six closed channels with
reservation3 and unchanged connected state. All eight scopes are inactive; three
compositors and six original guardian receipts close successfully.

M309 supersedes M308's adoption-before-binding owner-loss limitation only in the
same physical compositor. Partial/early initial startup, interrupted provisioning,
compositor/logout/reboot recovery, compatible-generation preflight and public
profile workflows remain. Native qualification uses synthetic Eyre, a private
keyring and headless Sway, not the installed full shell or Hyprland. Production
plugins/services and protected or tmux/temp-ur ships were untouched. Evidence
exports exclude keyrings, account/application state, piers and raw credentials.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-closed-initial-binding
