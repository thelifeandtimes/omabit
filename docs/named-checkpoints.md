# Named checkpoint plans and historical reports

This issue19 source slice passes focused race, real frozen-predecessor metadata,
current CLI/official-SDK parity, four-protocol and vet checks. See the
[scoped source evidence](diagnostics/2026-11-19-named-checkpoint-interfaces-source.json).
Matching artifacts and installed qualification remain pending; no test executor
or installed interface changes. The separate frozen12f78952 app VM does not
qualify these later source edits.

`tests.plans.v1` and `tests.plan.v1` read two compiled plans:
`runtime-lifecycle-v1` and `runtime-http-v1`. They expose the existing local-owner
runtime/HTTP qualifier's pinned image/pill, resource requirements, limits, network,
steps and remaining gates. Their legacy `urbit-runner runtime-plan`/`http-plan`
JSON payloads use the same constructors and retain their existing shape. The
version1 shared envelope adds a descriptor and digest. Reading a plan performs
no setup, download, registry read or runtime operation. `executable_here=false`
and `runtime_qualified=false` are explicit.

```sh
omabit call tests.plans.v1
omabit call tests.plan.v1 --input-json '{"id":"runtime-http-v1"}'
```

These operations do not execute tests through CLI or MCP. Existing explicitly
acknowledged qualifiers remain separate local-owner commands, subject to their
own host/management and disposable-runtime admission. Collector enrollment grants
no such authority. A future executor must bind the exact source, environment,
effects, limits and durable outcome; neither a plan digest nor descriptive
`DeskConfig.Checks` strings establish that execution occurred. No generic command,
shell, Dojo, SQL, Docker or caller-supplied check executor is exposed.

Historical report reads require a separate explicit startup selection:

```sh
omabit mcp serve --transport stdio \
  --allow-test-report SAVED:SHA256:/ABS/PATH/report.json
```

The same flag is available on an individual `omabit call`. Repeat it for at most32
reports. Each selected file must be an ordinary owner-private0400/0600 single-link
file, at a physical absolute path, no larger than1MiB. Its complete SHA256 must
match the startup selection. Links, FIFO/devices, unsafe permissions, changed
bindings, oversized or ambiguous/unsupported JSON refuse. The file is closed
before serving, and later replacement cannot change the immutable snapshot.
Selecting a source root or runner does not authorize report files. An operation
cannot supply a path, scan report directories or load another file.

Call `tests.reports.v1` with `include_private_data=true` to read the allowed
name/plan/digest bindings. Then select all three plus that explicit private-data
choice through `tests.report.v1`. A foreign name, plan or digest refuses. Both
interfaces use the same handler, including the local authenticated HTTP transport
when configured. HTTP credentials therefore cover the report snapshots explicitly
selected for that server; a runner allowlist does not narrow this separate local
data authority.

```json
{
  "name": "SAVED",
  "plan_id": "runtime-lifecycle-v1",
  "sha256": "EXACT_LOWERCASE_SHA256",
  "include_private_data": true
}
```

The reader currently accepts the existing `runtimequalify.Result` report shape
for these two profiles only. It returns reported phase outcomes/counts and
checkpoint/cleanup/limits facts, the selected content digest, snapshot read time
and a comparison of reported image/pill/engine pins to the current compiled plan.
Raw logs, free-form phase names/errors, host paths, network addresses and container
IDs are omitted. Incomplete/failed and different-pin reports stay visibly reported
as such. Contradictory success, production-qualification claims and unsupported
shapes refuse; the reader never edits the original to make it acceptable.

A content digest authenticates selected bytes. Anyone able to write a report
could write its outcome fields, so `reported_checkpoint_passed` is not independent
attestation. These legacy reports do not establish a repository snapshot or
execution environment. The response states that provenance is unavailable,
`executed_by_this_service=false` and `runtime_qualified=false`. A matching pin or
current plan digest does not turn old or caller-supplied evidence into a pass for
the current source. Native VM wrapper reports, arbitrary application repository
checks and full qualification-report execution are separate future contracts.

This additive reader changes no runner database schema, saved report bytes,
runtime/recording format or installed state. Generated CLI help/reference and
matching packaging remain final gates. One initial test incorrectly required
empty stderr; correct it to permit documented SDK diagnostics while refusing
private report-content/path leakage. The unchanged product passes that corrected
test. Source evidence is separate from actual named-client/provider connections
and execution of named qualification plans, which remain unavailable here.
