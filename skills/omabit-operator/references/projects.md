# Review a fake-project plan

For the distinct private keyed chain profile, use
[private-chain review](private-chains.md). The fake registration/lifecycle commands
below do not accept private-chain plans.

Use the intended collector generation. `help projects` and `project-plan --help`
are offline discovery; they do not read enrollment or contact a runner. If the
selected build lacks them, report that version limit instead of changing installed
tools or inferring project support from another build.

Specify the complete set of fresh fake galaxies and unique management names:

```sh
urbit-collector --format text project-plan --label demo \
  --member demo-zod:~zod:18381 --member demo-nec:~nec:18382 \
  --shared-administration
```

The HTTP ports are proposed unprivileged runner-loopback ports. Internal HTTP ports
are assigned from the sorted complete member set; changing that set changes the
plan. `--shared-administration` acknowledges that members share administrative
loopback access. Use this topology for trusted development apps.

Omit `--runner`, `--socket` and `--expected-runner-id`: this command refuses runner
selection. JSON is the default and reports the canonical `plan`, `review_digest`,
`reservation_check: "not-performed"` and `runtime_authorized: false`. Preserve the
whole reviewed plan with its digest. The digest is not a runtime project ID.

Planning does not check the host's actual port/instance reservations or create
resources. Registration is a separate explicit step, described below; initial runtime
actions have separate capabilities, described below. A private test pass
does not establish an installed runner's capabilities. Do not translate a plan
into Docker commands or use ordinary instance Start as a substitute for a project
operation. Later runtime admission requires the intended runner's capability and
fresh resource checks within the user's authorized scope.

For saved runner reservations, select the enrolled runner separately:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID projects
```

`project-reservations-v1` advertises this read-only view. It validates the stored
project/member claims and reports `runtime_observation: "not-performed"`; a reserved
port is not evidence of a running endpoint. Older runners may lack this capability.
Reading reservations does not grant project registration or lifecycle support.

For a project's saved operation history, keep the same runner pin and use the
project ID returned by `projects`:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID project-status PROJECT_ID
```

This separate read requires `project-saved-status-v1`. Older runners lacking it
can still provide their supported inventory and `snapshot`; do not infer that an
unsupported status route means no project exists. `project-status` returns the
newest 128 saved receipts in insertion order, with `has_older_operations` indicating
that earlier history exists. Parent recovery and observation/continuation attempts
retain their individual actors and request IDs.

The read timestamp dates the database snapshot, not a Docker inspection. A saved
`running` or `succeeded` operation does not prove current process health or Eyre
readiness. An `interrupted` receipt is not permission to repeat its action. Retain
its actor/request/operation identity for explicit reconciliation; do not mint a
new request to bypass uncertainty. This status command does not resume anything.

## Reserve a reviewed project

`project-register --help` is offline. Registration requires the selected runner's
`project-registration-v1` capability and operator role. Save the full JSON from
`project-plan` to a regular file, review every member/port and preserve its digest.
Do not save text output as the plan. Use the existing task authorization for this
reservation; acknowledgement flags do not require asking permission again.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID project-register \
  --plan /absolute/reviewed-plan.json --review-digest REVIEW_DIGEST \
  --request-id ORIGINAL_REQUEST --confirm-reservation
```

This atomically reserves the complete fresh-fake member IDs and runner-loopback
ports. It checks host port availability but creates no instance, pier, container or
running ship. The response returns its project ID and durable registration receipt.
An existing standalone ship cannot join through this command. Read `project-status`
with the returned ID; runtime launch is a separate explicit operation.

After a lost reply, preserve the runner identity, operator, full plan, digest and
request ID. Run the same command with `--replay-only` to retrieve that original
receipt. The lookup cannot create a missing reservation and does not recheck host
ports. A changed request or another actor is refused. If it returns
`replay_not_found`, review the unresolved original attempt before deciding whether
any new effect is authorized; do not automatically remove the lookup flag or mint
a new request ID. Never treat a saved reservation as current runtime health.

## Initial runtime actions

Read `project-operate --help` before using a runner's separately advertised action.
The unreleased source exposes these initial actions; installed older runners may
not support them. Keep each action within the authorized disposable test boundary.

| Action | Required capability | Effect |
| --- | --- | --- |
| `prepare` | `project-prepare-v1` | Create the private bridge and stopped namespace anchor. |
| `start-anchor` | `project-anchor-start-v1` | Start that prepared anchor; no ship boots. |
| `prepare-storage` | `project-member-storage-v1` | Create one reserved member's exclusive storage directory. |
| `launch` | `project-member-launch-v1` | Boot one fresh member after anchor/storage prerequisites. |
| `observe` | `project-member-observe-v1` | Record initial identity/readiness; no runtime action. |

`prepare-storage`, `launch` and `observe` require `--member INSTANCE`. Complete
launch for the whole member set before initial `observe`. Use the saved project ID,
revision and digest from its reservation, and a distinct request ID for each
separately authorized action:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID project-operate \
  PROJECT_ID prepare --revision 1 --review-digest REVIEW_DIGEST \
  --request-id PREPARE_REQUEST --confirm-action
```

The response is an operation receipt, often initially `queued`. Poll
`project-status PROJECT_ID` to observe its saved status; success is not a fresh
health guarantee. A timeout or lost response requires the same inputs with
`--replay-only`; it returns the original receipt without queuing a worker. An
interrupted result stays interrupted. Do not manufacture a new ID, choose another
action, or repeat Docker commands to bypass it. Coordinated anchor recovery uses the separately reviewed `project-recover` command.

Source and inert end-to-end CLI checks are recorded separately from actual Docker
qualification. A capability advertises implementation support; it does not replace
the user's scope, exact runner selection or fresh runtime admission.

## Member lifecycle receipts

The following explicit actions also use `project-operate`. Each requires
`--member`, `--after`, the reviewed revision/digest, a distinct request ID and
`--confirm-action`. A prerequisite is a saved operation ID for that exact member;
it is never a container ID, ship name or filesystem path.

| Action | Required capability | `--after` receipt | Effect |
| --- | --- | --- | --- |
| `stop` | `project-member-stop-v1` | Initial, resumed or current readiness | Signal only the observed member to stop. |
| `observe-exit` | `project-member-exit-v1` | Stop | Observe exit without repeating the signal. |
| `retire` | `project-member-retire-v1` | Exit observation | Remove that exited container; retain its pier. |
| `resume` | `project-member-resume-v1` | Retirement | Create/start a successor using the same pier. |
| `observe-resumed` | `project-member-resumed-observe-v1` | Resume | Record fresh replacement readiness. |
| `observe-current` | `project-member-current-observe-v1` | Exact current creator | Record current whole-project context. |

`observe-current` additionally requires `--original-readiness` naming the initial
initialized-pier receipt. Its creator is the exact launch/resume/recovery member
creator from history. Keep that original receipt through later replacements.
Peer replacements can make an earlier readiness context stale, requiring a fresh
current observation before Stop or later lifecycle actions. These selections do
not waive current ownership, clean exit, storage or project-wide recovery checks.

Use `project-status` to read saved receipts. A queued/succeeded record alone does
not prove present health. After a lost response, preserve every input including
both prerequisite fields and add only `--replay-only`. Missing or mismatched
prerequisites are refusals to correct after reviewing history, not an instruction
to create a replacement receipt. An interrupted operation needs its own explicit
recovery path; this command does not automatically retry effects.

These public commands remain development source pending public-command Docker
qualification and UI integration. Do not update a production runner to try them.

## Inspect an interrupted recovery checkpoint

Use `project-status` to identify the owning `recover-project-anchor` receipt, then:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  project-recovery-checkpoint PROJECT_ID RECOVERY_OPERATION_ID
```

This requires `project-recovery-checkpoint-v1`; observer access is sufficient.
The response includes the saved phase/sequence, plan/history/checkpoint digests
and a next-step hint. It performs no Docker inspection, login or runtime action.
A hint is not authorization and a saved successful recovery is not current health.

- `pending`: retained project work is active; do not schedule a duplicate.
- `observation`: an uncertain effect needs a separate read-only recovery observation.
- `continuation`: the saved checkpoint permits considering a new explicitly reviewed
  continuation request; it does not resume a worker automatically.
- `review`: preserve the receipt and inspect the unresolved state. In particular,
  Create with no returned ID does not authorize adoption or another Create.
- `complete`: that saved recovery completed; this read has not checked current health.

Keep the exact parent, sequence and digests for any separately supported recovery
request. Use the separate `project-recover` command only within the authorized scope;
do not substitute raw Docker commands or bypass the retained journal.

## Select saved history before a new recovery

`project-recovery-source PROJECT_ID` on the same pinned enrolled runner requires
`project-recovery-source-v1` and allows observer access. It returns the reviewed
project digest, current saved history digest and each member's exact creator,
generation, original-readiness receipt and available current-readiness receipt.
Initial launches/readiness must already exist. Missing prerequisites are refusals,
not a reason to recreate the project or repeat earlier effects.

This is a database read, not a runtime health or ownership probe and not permission
to begin recovery. For an existing interrupted recovery, keep using its exact
`project-recovery-checkpoint` parent and preserved source. Never substitute a new
history digest to bypass its uncertain outcome. Public recovery actions remain
separate from these read-only selection commands.

## Request coordinated recovery explicitly

`project-recover PROJECT_ID start|observe|continue` requires a pinned enrolled
operator runner. Read its `--help`; supply `--revision`, `--review-digest`,
`--history-digest`, a distinct `--request-id` and `--confirm-action`.

| Action | Capability | Selection and effect |
| --- | --- | --- |
| `start` | `project-anchor-recovery-v1` | Use the current saved recovery source. Reviews runtime ownership, then drains members, recovers the namespace and replaces containers on the same piers. May interrupt every member. |
| `observe` | `project-recovery-observe-v1` | Use an interrupted parent's exact checkpoint. Observe one uncertain outcome and record evidence without repeating its effect. |
| `continue` | `project-recovery-continue-v1` | Use an eligible interrupted parent's exact checkpoint. Authorize new work from that checkpoint. |

`observe` and `continue` also require `--recovery OPERATION`, `--sequence N`
(including an explicit zero when applicable) and `--checkpoint-digest HASH` from
`project-recovery-checkpoint`. Keep that parent's preserved history digest; do not
replace it with the newest recovery-source digest. Saved hints grant no authority.
Unknown-ID Create remains review-required: no command adopts a container or
repeats an uncertain Create. Admission may refuse stale or ineligible selections.

After a lost response, preserve every input and add only `--replay-only`. This
returns the same actor-scoped receipt and never creates a missing one or schedules
another worker. An interrupted result stays interrupted. A separately reviewed
observation/continuation has its own request ID and receipt. Follow those receipts
with `project-status` and the parent with `project-recovery-checkpoint`; successful
saved work is not a fresh health check.

These commands are development source awaiting public-command Docker qualification
and UI integration. Production rollout remains outside the experiment scope.
