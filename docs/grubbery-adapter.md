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
