# Application testing and recovery

Bind evidence to the source snapshot, tool build, selected runner, project/member
generation, runtime/pill, policy/workflow revision, app API/state version and
artifact hashes. Keep private process/pier bindings and secrets outside Git.
Configured check names are descriptive; externally supplied release evidence must
identify who ran which reviewed check against which input/environment. It does not
prove that Omabit executed a test. Never weaken checks to obtain a passing label.

Use one disposable runtime experiment at a time within its admitted storage and
CPU/memory limits. Declare scratch before launch, measure allocated bytes before
and after, reuse immutable inputs, and keep compact results. Retire only classified
redundant generated outputs with manifests; never prune an existing pier or manual
review fixture to make space. Confirm excluded services and piers from the recorded
environment boundaries without inspecting them.

Separate these claims: local source tests, native Hoon compilation, live
single-ship app behavior, peer discovery/messaging, browser/client behavior,
same-pier restart and non-destructive upgrade. Report failures and limitations at
the boundary where they occur. A synthetic observer is not live runtime evidence;
a headless served-byte check is not an interactive browser test.

Save every reviewed plan and original request/operation ID before submission.
Queued is not complete. On a lost reply, timeout or process loss, use saved status
and exact lookup/reconciliation through CLI or MCP; preserve sealed JSON integer
bindings without numeric rounding. Do not mint a new request to escape uncertainty.
Cancelling a wait or closing an MCP process does not cancel runner-owned work.

For an expanded-member interrupted Stop, saved diagnosis and `--probe` are
different reads. A verified clean-stop review may be applied through
`project.stop-reconcile.v1` or the matching collector command. It preserves the
original interruption and adds a linked audit receipt; ordinary Start is separate.
Nonzero exit, OOM, stale evidence, ownership drift or an unresolved older operation
must refuse this clean-stop path. Hand other phases to supported operator recovery;
never edit SQL, clear fences or repeatedly signal a ship to finish a test.

Before finishing, account for all uncertain effects, temporary mount ownership and
cleanup. Preserve raw owner-only runtime evidence according to the recording
contract; share only an explicitly selected derived/redacted export. Do not paste
codes, cookies, raw authentication responses or private logs into reports.
