# 0.3.5 source and distribution review

Reviewed frozen source `d3979dbb30efc344ac35687ba2782662981ee478` against
0.3.4 master `31bdea8`, with the prior performance measurements and regression
evidence. This is the release agent's review, not an independent human audit.

- Transport reuse remains private to each enrollment/client lifetime and bounded
  to three connections. New connections still perform identity checks. Capability
  evidence is shared only within one observation; response identities and live
  runner authorization remain checked. Mutations are not replayed on reconnect.
- Schedule control accepts bounded NDJSON interval/refresh input, with no runner,
  credential or action selection. Visibility changes preserve the collector and
  transport. Pending operations retain fast observations; cancellation and EOF
  join the input reader and close transports.
- Panel ownership is tracked independently so closing one view cannot idle a
  remaining open view. Hidden clocks update on observations and view opening.
  Histories preserve runtime replacement, missing-sample gaps and size limits.
- Bootstrap selection observation reuses a leased process but revalidates state
  and pins each time. Activation/recovery continues through existing locks,
  archive verification and readiness checks. A failed candidate retains the prior
  usable desktop, verified by actual native QML tests.
- Snapshot history batches select the latest 128 completed roll records per
  instance before decoding runtime metadata, preserve current-runtime binding and
  per-instance order, and keep the original single-instance query. No database
  migration or persistent-format change is introduced.
- Stable packaging was rebuilt from a clean commit. Its backend source digest
  matches the RC; only the three product version strings differ. Final pins name
  exact immutable assets, and the previous 0.3.4 artifacts remain unchanged.

No additional source defect was identified in this review. Exact stable service
upgrade, final release-policy validation and protected source/race/QML CI are
separate gates recorded in the qualification/publication evidence. Existing
battery, remote-host, broader security/privacy and long-duration limits remain.
