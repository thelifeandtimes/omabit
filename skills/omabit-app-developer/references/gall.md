# Gall application development

Inspect the actual desk, agent names, marks, paths/wires and persisted state
versions. Record the intended publisher and exact runtime/pill inputs. Treat
the app's configured API as authoritative; there is no universal readiness route,
install publisher or Hoon test command. UI ownership is optional: a standalone
browser game can use direct authenticated Eyre without an Omarchy frontend.

The current named backend sample is available with
`omabit call app.sample.v1 --input-json '{"name":"gall-state-v1"}'` or MCP tool
`omabit.app.sample.v1`. It returns five bounded source files, exact staging/input
hashes, API/state expectations and its explicit current qualification status.
Save only the source files into a fresh staging directory and follow its manifest;
do not use an entire source checkout as a desk. Sample reads run no tests or
deployment. Missing local sample support requires matching tooling, not a guessed
download or shell installer.

Read [the common workflow](app-development.md) before deployment. Resolve the
current project generation, desk workflow and release configuration. A desk's
existence does not prove that an agent exists, compiled, is active or speaks the
expected API. Distinguish absent/suspended installation from authentication,
permission and protocol-version failures. Preserve Urgit bindings when selected;
the framework choice does not change source ownership.

Keep app source, custom marks and the minimal dependencies in a bounded staging
directory outside any live Unix mount. `desk-plan-stage` and `desk-stage` transfer
frozen inputs; `desk-review-commit`, `desk-plan-commit` and `desk-commit` bind the
accepted delta to current source/runtime/policy. These operations create no Unix
source mount. Save complete stage/plan IDs and receipts. Keep generated and unowned
desk paths separate from explicitly owned app paths; do not prune the scaffold.

Observe source revision/hash independently from `desk-app-status` and the
app-specific read/poke/watch contract. A commit receipt verifies its defined source
effect, not arbitrary Hoon compilation or behavior. Activate only through the
reviewed release plan, with the appropriate Commit/Publish gates. Frontend upload
requires its own exact artifact hash and independently checked served bytes.
Treat caller-supplied release checks as external provenance, not jobs run by Omabit.

Qualify a minimal app by compiling the exact native Hoon source on a fresh fake,
reading initial state, submitting a declared marked poke and observing a watch
fact. Reload its code, then restart the same disposable pier through supported
lifecycle operations. Verify persisted values, no duplicate writes and
subscription re-establishment using the same API/wire contract. A frontend reload
does not test a ship restart. For peer behavior, bind exact ship/member identities
and verify both sides, rather than deriving endpoint ports from member order.

State migration belongs to the application. Keep old tagged state fixtures and
test supported on-load transitions; refuse unknown versions deliberately.
Changing client assets is not a rollback of Gall state. Separate source/API/state
compatibility and never use a pier reset as an application rollback procedure.

If compilation/activation is slow or its response is lost, inspect the saved
operation and application stages. Reconcile the original operation before another
effect. Report unresolved native dispatch explicitly; do not conclude rejection
from an HTTP error alone. See [testing and recovery](testing-and-recovery.md).
