# Versioned Grubbery adapter development

Issue18's adapter targets `nisfeb/grubbery` commit
`7f7701d546ce115eea5f37f66465b362cb121774`. The historical exercise at
`ff2b66b58d1ed15f4aecbe7a065559028eb1fb7b` is a different contract. The new
pin's documentation embeds older live-code references, so the pinned Hoon source
is authoritative for this adapter. Source review and exact-byte checks do not
qualify native installation or application behavior. Grubbery preflight continues
to report `missing-workflow-adapter` until those gates pass.

The shared offline `app.framework.v1` operation returns the compiled definition
`grubbery-7f7701d-v1`, upstream commit, MIT notice and complete Clay desk manifest:
1443files,8013304bytes, source digest
`7d2ae430acd4868bb56268188dff57addcdf4520abcc7aabaa6b9a342b22febc`.
Its authored qualification is `native-workflow-pending`. Omabit embeds only
metadata, not a framework installer, download or hidden framework source tree.

After separately obtaining the exact public source, verify only its `desk`
subtree outside live mounts:

```sh
omabit call app.framework.v1 \
  --input-json '{"name":"grubbery-7f7701d-v1"}'
omabit call app.verify-framework-source.v1 \
  --allow-source framework:/absolute/grubbery-checkout \
  --input-json '{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"desk"}}'
```

Both operations use the same handler through stdio MCP, with the
`omabit.` tool prefix. Source authority must be explicitly selected at startup;
tools receive a name and relative subtree. Missing, changed or additional files
refuse. Links and special files refuse through the same confined snapshot rules
as staging. Verification never creates a registry, contacts a ship, grants policy,
stages source or performs an installation. The existing staging plan and transfer
independently bind/recheck the digest; this earlier check is not transfer authority.

The frozen [kernel](https://github.com/nisfeb/grubbery/blob/7f7701d546ce115eea5f37f66465b362cb121774/desk/app/grubbery.hoon)
and [ball API](https://github.com/nisfeb/grubbery/blob/7f7701d546ce115eea5f37f66465b362cb121774/desk/lib/ball-api.hoon)
use `/grubbery/api/peek`, `/poke`, `/make`, `/cull`, `/sand` and `/keep`.
Directory paths have a meaningful trailing slash. Build, bang, queue and process
observations are separate kernel reads. The old exercise's `/grubbery/ball?list=1`
is not this adapter's API. A source commit label or a successful upload does not
prove this running kernel's bytes or a loaded app's state.

The pinned [desk nexus](https://github.com/nisfeb/grubbery/blob/7f7701d546ce115eea5f37f66465b362cb121774/desk/gub/nex/desk.hoon)
syncs a code namespace and uses a version grub as a change signal, not historical
source pinning. Code, loaded instance and retained state therefore need independent
checks. The [shell](https://github.com/nisfeb/grubbery/blob/7f7701d546ce115eea5f37f66465b362cb121774/desk/gub/nex/shell.hoon)
records authoritative approvals separately from app-owned `grant.json`. Approval
replaces the weir with the selected declared subset; reload is a separate request.
Built-ins under the trusted root differ from initially closed desk applications.
An adapter must retain that separation and must not install a new app as a trusted
built-in to avoid its permit gate.

Still required: native compilation/bootstrap and timeout recovery, supported
policy-enforced app deployment, discovery and alias-conflict handling, exact
permission/route/served-artifact checks, resumable uncertain-write receipts,
state-preserving update/restart, two-ship discovery and the three-member workflow.
No live Grubbery capability, API effect or completed issue18 claim follows from
the offline contract. Unix source mounts remain unnecessary for this source check.

The first native bootstrap capsule stopped at a correct ownership conflict for
`mar/hoon.hoon` and `sys.kelvin`; no framework commit or activation was submitted.
The fresh kernel shut down cleanly. See the [original failure record](diagnostics/2026-11-17-grubbery-bootstrap-first-capsule.json).
This pin also reads its own Clay source from the literal desk `grubbery`; the
separate fresh-only gate uses that required name inside its private ship.

Shared `desk.export-source.v1` provides an operator-only, bounded read of explicit
files at the complete `observation` returned by `desk.inspect.v1`. Pass that
unchanged observation and sorted unique `paths`. Up to64files and256KiB are
returned as canonical `content_base64` plus SHA256, preserving binary and empty
content. The runner brackets mark-to-MIME conversion with exact process, pier and
Clay baseline checks. These bytes may use the mark's canonical spelling; they are
private source, not a redacted/shareable diagnostic. The read changes no ownership,
policy, mount or source. Stage only the original conflicting files and explicitly
review/adopt their equality through existing adoption gates before planning the
complete framework replacement. Never adopt a changed framework file as if it were
the original scaffold. Native qualification of this continuation remains pending.

The [native bootstrap continuation](diagnostics/2026-11-17-grubbery-native-bootstrap.json)
now passes for those exact inputs. It proves the original-scaffold read/adoption,
verified full source commit, one cold activation with a lost reply resolved only
by observation, live declared agents, authenticated kernel diagnostics, no mount
and clean shutdown. The root `/code` report has1335files and0build failures.
The runtime separately reports4failures in `/apps/shell/tools/code`; that nested
namespace was not individually inspected by that bootstrap case. This supersedes only the preceding
continuation deferral. The remaining app/permit/route/served-byte/restart/peer and
installed gates still apply, and preflight remains unavailable.

The internal coordinator now has a schema61 SQLite persistence adapter: immutable
headers and append-only events, transactional exact leases, original observed
marker ancestry, replay without source/runtime access, and preactivation
abandonment that preserves staging and live state. [Source migration checks](diagnostics/2026-11-18-grubbery-app-store-migration.json)
use actual original60 and retained59 fixtures. A real process exit after dispatch
commit preserves that original fence on reopen. These gates qualify the journal,
not native namespace observations, public app API or a working Grubbery app.

The later [first native app contract case](diagnostics/2026-11-18-grubbery-app-contract-first-capsule.json)
confirms a single file make creates its plain parent and marker, and the tiny
local state nexus compiles with the framework-generated marks. It fails source
following with a null version: the draft marker omitted the `version` value that
pinned +version-text requires. Preserve that failed result and exact old plan;
correct it through an explicit successor generation before app dispatch.

The same kernel diagnostic identifies the four nested failures: `lib/pytz.hoon`,
its dependent `lib/rules.hoon` and `lib/calendar.hoon`, plus the incompatible extra
`to-lines` arm in `lib/tools/read-terminal.hoon`. They remain unavailable in this
pin. This is not permission to alter the pinned framework or treat those tools
as working. State updates, permits, routes, served bytes, restarts and peers still
need their own qualification; source compilation does not establish them.

The separately supervised [corrected native contract](diagnostics/2026-11-18-grubbery-native-app-core.json)
passes with the original failure retained. It proves the local state nexus,
explicit version following, saved-state poke/read, state-preserving source update,
new code label and a duplicate-create rejection with unchanged original marker.
The ship exits cleanly. This is kernel-only evidence with synthetic lifecycle;
it does not qualify the public managed adapter, permits, local aliases, HTTP
routes/assets, runtime restart, peers, installed upgrades or DEV deployment.

The development coordinator now derives native source expectations from the
complete immutable namespace history in database creation order. It refuses
unowned roots, changed markers, foreign source paths and changed foundational
bytes. Virtual foundational grubs are verified separately from the materialized
tree. They do not establish a materialized `mar` directory or waive its original
directory phase; native custom-mark semantics remain under qualification.
No arbitrary existing directory is adopted. Source bytes, compilation and the
followed live mirror are independently checked. Native reads stop at512 calls,
the existing1MiB response/4096-node tree limits and the caller's deadline;
exceeding a bound refuses without pruning history or resubmitting an effect.

The runner factory rechecks authenticated operator access, management fences,
configuration/container/pier, the original verified framework commit, separate
observed activation, live declared agents and current application policy. Its
new-effect admission requires Commit; read-only recovery remains available after
Commit revocation. An unfinished app operation protects its private stage from
the existing discard API. Owner-session cleanup must be confirmed before native
evidence is accepted; unconfirmed cleanup cannot fabricate an acknowledgement.

These are source and inert fixture gates, including actual SQLite reopen with
lost reply, missing cache and revoked Commit. They do not qualify the native
coordinator or resolve durable authentication cleanup after process death.
Public app dispatch, permits/publication, routes/served artifacts, restart and
peer tests remain required. Grubbery capability is still unavailable.

The first actual coordinator run reaches an
[empty-code-tree observation mismatch](diagnostics/2026-11-18-grubbery-coordinator-empty-code-tree.json).
Framework bootstrap and the original marker/release-parent proofs pass, but a
newly created code namespace is empty. The source/inert model's eager foundational
files are not a native fact. The acknowledged phase remains uncertain, no next
phase is submitted, and the ship stops cleanly. A separate fresh kernel layout
case now records source-tree/raw-byte/build/version/mirror boundaries before
changing that projection. Previous generated-directory assertions remain
unqualified; no live Grubbery capability follows from these failed native gates.

The [source-layout contract](diagnostics/2026-11-18-grubbery-native-source-layout.json)
now passes: four exact pinned source grubs are directly readable while absent
from both materialized release and live-mirror trees. The corrected coordinator
keeps those observations separate and restores one native call per exact phase.
Inert/race checks pass for that separation and for refusing a newly introduced
materialized directory even with matching pinned source bytes. The corrected
managed coordinator still needs its own fresh native gate; kernel diagnostics
alone do not qualify it.

The corrected [private managed native gate](diagnostics/2026-11-18-grubbery-managed-native-source.json)
now passes all12 exact phases against the pinned kernel, including durable
SQLite dispatch, independent ownership/source/compile/followed-mirror checks and
confirmed per-call session cleanup. The fresh ship stops cleanly. A separate
kernel diagnostic proves a custom-mark directory needs its explicit single
create before the custom source compiles alongside four virtual foundations.
This supersedes the preceding managed-native deferral only for frozen ac3eec2.
It does not qualify product Docker/curl, public interfaces, managed second-version
state/restart, process-death cleanup, permits, routes/assets, peers or installed
upgrades. Live Grubbery capability remains unavailable pending those gates.

The typed public source endpoints now have source/inert qualification for review,
begin, one exact phase, saved receipt, read-only reconcile and safe abandonment.
Actual SQLite/HTTP tests preserve a lost reply, recover with missing stage cache
and revoked Commit, and prove old-phase replay never selects the next phase.
CLI and official-SDK stdio return identical results for all five calls. Restricted
SSH tests cover larger bounded plans, missing capability, observer exclusion and
unchanged ordinary request limits. These interface gates do not extend the frozen
ac3eec2 native result to newer product artifacts or Docker transport. Original
actor/process/config binding remains required for unfinished recovery.

The [frozen d2d8cf9 public native gate](diagnostics/2026-11-18-grubbery-public-native-workflow.json)
passes the actual runner source API and pinned registry/client against a fresh
pinned kernel. Both source versions are independently compiled and followed;
saved state survives the update, the new code label is observed, an exact owned
same-pier restart is clean, and another state write succeeds. Both original
SQLite receipts survive reopen and old-phase replay unchanged. Each original
phase is dispatched once. This uses synthetic lifecycle and direct native owner
HTTP, so product Docker/curl, pending-operation process rebinding, durable owner
cleanup on process death, the native CLI combination, permits/routes/assets,
peers and installed upgrades remain unqualified. Full preflight stays unavailable.

Maintained `grubbery-state-v1` and `grubbery-state-v2` samples now return the exact
two-file source bytes from the frozen public native workflow through
`app.sample.v1`. Their separate definition/evidence metadata records that named
packaging was added afterward; it does not relabel the older artifact as a named
sample test. Source digest, pinned framework/runtime inputs and qualification
limits are explicit. Both stage without metadata/notices; v2 changes the code
label while preserving one saved JSON string. Pure source/native-fixture equality
and actual CLI/stdio sample parity pass. Sample state probes remain test-only;
there is no new public state read/poke/watch capability or complete preflight.
## Bounded read-scope development checkpoint

The real Docker/curl product case now resolves framework activation by read-only
reconciliation, without another install. Five original app source phases are
observed; the sixth is acknowledged without its final observation, retaining the
exact lease and all original receipts. The overall case remains failed. Its
last call lasts about241seconds, consistent with the four-minute bound; the old
fixture loses its CLI error envelope, so that precise error is not proven. See
the [partial product evidence](diagnostics/2026-11-18-grubbery-product-partial-source.json).

A source follow-up authenticates once for one bounded read-only capture, instead
of once for every verification read. It preserves sequential/conditional source,
tree, marker and mirror checks and their original evidence digests. Each request
keeps its exact binding, process, URL/method/body and owner identity checks;
framework/pier checks bracket the capture, and normal runner policy/lifecycle
guards still bracket admission and dispatch. The callback exposes only typed
reads. Provisional `scope-active` responses are not evidence until logout is
confirmed. Cleanup failure, callback failure, drift or an invalid callback cannot
produce a proof. Retained readers refuse after the capture or parent lifetime.
Native writes retain their existing separately authenticated exact dispatch.

Source race tests compare full plans, read counts, conditional refusals and native
write counts with the original adapter. Inert transport tests cover one login/code
read/logout per capture, exact cookies, cleanup/callback failures, foreign actor,
parent cancellation, process/framework drift and retained-reader refusal. Existing
managed app source/journal regressions and focused vet pass. Native performance,
complete product workflow and pending-phase container/key recovery remain to
qualify. No capability or persistent format is promoted by these source gates.
The next product fixture retains private request/response JSON and bounded CLI
stderr before reporting a typed failure, preventing another lost error envelope.
