# Project allocation observations

The development contract `project-allocation-read-v1` adds an operator-private,
read-only census for a saved expanded project. It does not change persistent
formats or confer lifecycle authority. Source/interface tests and matching native
qualification are separate; see the [issue gates](issues-16-21-next-gates.md).

Read saved project status first, then select its exact revision and plan digest:

```sh
omabit --config-dir PROFILE call project.allocation.v1 --input-json \
  '{"name":"DEV_ENROLLMENT","runner_id":"RUNNER_ID","project_id":"PROJECT_ID","expected_revision":1,"review_digest":"SAVED_64_HEX_PLAN_DIGEST","include_private_data":true}'
```

Replace all example values with the selected enrollment and saved status. The
same typed operation is exposed as `omabit.project.allocation.v1` through MCP.
Its runner endpoint is `POST /v1/projects/allocation`. The selected Hello must
advertise the capability and operator role. Calls require explicit private-data
consent because results include exact storage paths and inode identities. No
request path, shell command, Docker command or caller-provided prior measurement
is accepted. A reservation without an expanded-member ledger is explicitly
unsupported by this first adapter; it is never an empty project measurement.

The server loads and hashes the saved ledger, validates current member creation
contracts, then checks the same ledger again after the filesystem read. The
response binds runner/mode, project, revision, generation, plan and ledger digest.
Concurrent ledger changes withhold the result. An interrupted operation remains
interrupted, retaining its original receipt and fence. The census neither probes
the engine/process/HTTP endpoint nor starts, stops or repairs anything.

Each declared root is labeled separately: current member mount parent, that
member's conventional private runtime cache, or the project's private resource
cache. Stored member mount/pier identities are checked before and after reading.
Cache identities are observations, not persisted lifecycle authority. Retired or
historical member storage, shared engine images/content, the global management
journal, staged application source and unrelated storage are excluded explicitly.
`shared_allowance_status` remains `not-configured` and
`full_project_budget_known` remains false. A successful census does not establish
the project's full budget or account shared components as zero.

`logical_bytes` sums regular-file inode lengths. `allocated_bytes` sums inode
blocks times512, including directory and special inode blocks. Internal hardlinks
count once for bytes and separately for directory entries. Sparse files can have
very different logical and allocated sizes. Shared extents, compression and
filesystem metadata mean inode-block accounting is not exclusive physical usage
or a reservation of available free space. These are non-atomic metadata censuses,
not event payload counts. No content of the measured files is read.

The walker pins physical directories, owner, device/mount and birth/inode
identities. It inspects pipes and sockets without reading or connecting to them.
Symlinks, foreign owners, cross-mount traversal, links outside a declared tree,
observed replacement/change, missing paths, cancellation, overflow, or the finite
entry/depth limits yield an explicit unknown reason. Unknown observations expose
no partial totals or measured roots. Missing paths are never recreated or deemed
disposable. Failures to establish or revalidate the saved member/root binding
withhold the entire observation with a typed API refusal. One census may run per
manager at a time, with a30second request
deadline,65roots,65,536total entries and depth64. Filesystem calls remain subject
to the underlying filesystem's responsiveness. There is no background scan loop.

Growth is an exact byte delta, direction and elapsed nanoseconds between two
server-held censuses with identical bindings and root identities. The server
uses monotonic completion times and at most32 cloned project samples. Windows
shorter than1second retain the older comparable baseline; windows older than
5minutes are unknown. Unknown reads clear their baseline, binding changes replace
it, and a manager restart starts without samples. Neither the caller nor wall
clock timestamps supply prior measurement authority. Net inode/file growth is
not application-event growth or an automatic stop/resume decision.

The [start resource policy](resource-policy.md) remains a separate admission
check for currently available host RAM/disk. Project incident containment still
requires explicit budgets/shared allowances, a durable incident latch, reviewed
resume and journaled stops of freshly verified exact members. This read contract
adds no thresholds, automatic pruning, incident persistence or containment.
