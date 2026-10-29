# Built-CLI operator walkthrough

The built collector passed five scripted tasks against the built runner's in-memory
simulator. All processes ran inside a private Bubblewrap namespace with no host
HOME, network, runner socket, desktop IPC or engine. This is a CLI task checkpoint,
not real ship or runtime qualification. The source driver is
`scripts/test-operator-walkthrough.py`; [the evidence](2026-10-22-operator-walkthrough.json)
identifies both binaries, the driver and the complete transcript.

The 31-command walkthrough:

- Enrolled its own simulator, selected the returned runner ID and inspected
  capabilities and snapshots in JSON and readable text.
- Simulated another operator's configuration change, rejected stale Start requests
  in both formats, refreshed the snapshot and verified no operation was dispatched.
- Submitted the still-intended Start with the reviewed revision, retained its
  asynchronous receipt, followed that exact operation to completion and checked
  resulting state. Re-presenting the terminal request returned the original receipt.
- Stopped only the simulator process and checked that inventory reported unavailable
  with no current ship snapshot. The last-known running state did not become stopped.
- Started a different simulator at the same socket and verified identity mismatch
  without silently changing enrollment.

An unavailable inventory exits 1 while returning a structured partial inventory on
stdout. Callers must inspect that result; nonzero exit does not guarantee a simple
error envelope on stderr. The operator reference now states this distinction.

Text output was inspected through the recorded transcript and asserted for identity,
receipt and unavailable-state clarity. This does not claim a human usability study
or an independent agent evaluation. The real import rejection/recovery task,
transport-loss uncertainty, structured next actions, developer skills and packaged
skill distribution remain follow-up acceptance work.
