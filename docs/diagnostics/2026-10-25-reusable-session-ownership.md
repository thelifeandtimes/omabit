# Reusable session ownership and legacy writer exclusion

One format3 intent imports checkpoints1/2/3 through bounded ancestry. It preserves
the original closed cohort, exact journal prefix and new display binding. Before
claiming the runtime it publishes object-bound refusal records in both legacy
intent namespaces. Their original readers reject these shapes, preventing a
legacy writer from independently preparing a conflicting successor. Existing
writers are unchanged. New readers verify the owning intent and fence identities.

20 private tests pass. Actual format1/2 checkpoint imports are covered, as are four
lost publication replies, exact replay, missing/replaced fences, lost primary
intent, changed cohort/history/display and competing ownership. If a legacy writer
prepares during interruption before its fence is published, the new path refuses
without claiming the runtime or constructing applications. It does not steal that
ownership. Both fences must remain valid for eventual authority admission.

The initial harness count assertion caught42 cases because two temporary class
aliases exposed imported/duplicate tests. It stopped before execution. Removing
those aliases produced the intended20 cases; the failed launch is retained. Both
scopes are inactive, current and baseline source hashes match, copied intent and
checkpoint validate, and both original readers refuse generated fence bodies.

M334 resolves the preparation ownership gap identified in M333. It does not qualify
reusable authority or application startup. Repeated authority/inspection/cleanup
sources are staged separately and uninstalled. Next exercise three successive
sessions using one format family and unchanged package hashes. Claimed-but-unpublished
and pending/unbound loss remain separate gates. No production resources changed.
