# Public saved recovery checkpoint

The new `project-recovery-checkpoint PROJECT RECOVERY_OPERATION` command uses
`GET /v1/projects/<project>/recovery/<operation>` and requires the separate
`project-recovery-checkpoint-v1` capability. The CLI pins an enrolled runner;
observer access is sufficient.

It reads a consistent database transaction, validates stored causal history and
returns public project/receipt metadata, the saved phase/sequence and exact
plan/history/checkpoint digests. It omits private runtime paths, container IDs,
full journals and sessions. The response explicitly marks runtime observation
not-performed and action authority not-granted. No Docker backend is opened.

Next-step hints distinguish pending work, an uncertain outcome needing observation,
a checkpoint eligible for separately reviewed continuation, unresolved review and
historical completion. An unknown Create result stays review-required. No read
adopts a container, repeats an effect, changes an acknowledgement, writes a journal
or creates a worker. Completed history is not current ship health.

Managed inert tests pass98.549s across lost Stop/removal/Start, returned-ID Create,
unknown-ID Create, queued/completed recovery, observer route, missing selections,
private-field omission and corrupt mirrors. They compare database change counts,
backend calls and retained checkpoint equality. Client validation rejects changed
identity, project/receipt binding, authority/health claims, malformed digests and
invalid hints. Applied client1.028s, access1.008s and CLI1.027s pass with vet, diff
and operator-skill validation. Tested store/route bytes are applied unchanged;
only guidance was added after the first store test, so it was not repeated.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery-checkpoint`.
Schema40 and persistent receipts are unchanged. Frozen VM34 predates this read
route. Actual runtime/public-command qualification, initial recovery source review,
public recovery effects/observations/continuation and QML remain pending.
