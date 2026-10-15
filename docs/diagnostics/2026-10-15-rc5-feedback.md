# RC.5 feedback and supplied pill qualification

The requested changes cover explicit local setup software/source information,
the new Recommended Apps URL, stable log refresh controls, inline port lookup
progress/errors, visible public comet identities, HTTP fleet presentation and a
Coming Soon application-connections stub.

## UI and redaction verification

Mapped native-wrapper checks passed with deliberately slow log and port replies:

- Refresh logs retained its label, enabled state, button width/height and row
  geometry throughout an in-flight request. A delayed fixed-footprint indicator
  handles longer reads; repeated clicks share the in-flight read.
- Port lookup kept the form's height unchanged, rendered status beside the button,
  and exposed no global progress banner. Responses remain bound to selection/token.
- Setup exposes the three Omabit executable names, install locations and source/
  self-build links. Fleet rows show HTTP ports and Open actions with health gates.
- Application Connections renders Coming Soon and starts no application-session
  inventory/login or automatic theme-sync work.
- Full public comet names are extracted from `found comet` lines for early display.
  The authoritative runtime readiness/identity check remains separate.

Go tests preserve valid whole public comet names while filtering adjacent login
codes, boot keys and credential lines. Invalid/extended name tokens do not bypass
the existing filters. Full Go tests/vet, runtime/managed/transport race checks,
model, mapped onboarding/panel, installer and native-bootstrap tests passed.

## Supplied artifact

Source: `https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill`.

- Download: October15 at18:49:30UTC, with conditional ETag validation.
- ETag: `"f6be01d6c61667d844b65ae00d7fe92e-4"`.
- Last-Modified: October15 at18:04:58UTC.
- Size: **59,351,751 bytes**.
- SHA-256: **`85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`**.

The production Go pill resolver separately fetched the supplied URL and verified
the same bytes. The release attaches a frozen copy; the default remains the
user-requested S3 URL. Registration hashes the actual content selected at that time.

## Runtime smoke gate

On the existing sourceport qualification host, a new isolated fake `rc5-pill-check`
used the frozen pill, with `~zod` and HTTP18205. Its registration recorded the exact
hash/size above. Start succeeded at18:54:22UTC and verified healthy identity/HTTP;
clean Stop completed at18:54:23UTC. The previous two qualification piers remained
stopped and preserved. The new disposable pier is retained stopped.

The gate used the existing qualified owner-user runner/runtime, before RC.5
publication. It qualifies the new pill's basic runtime boot, not the full application
inventory or OTA behavior. The bounded log showed `%mcp-server` installation,
Grubbery's `23 files under /code did not compile` message and HTTP504 retries.

Private evidence: `build/rc5-input/input.json`,
`build/rc5-qualification/{source-resolve.json,start.json,pill-smoke.log}`.

Sourcebox's two user-created comets were running healthy at the pre-rollout
inspection, with no active operations. Deployment verification is recorded after
installing the frozen runner and cold-reloading the native plugin.
