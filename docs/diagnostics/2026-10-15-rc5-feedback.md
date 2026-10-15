# RC.5 feedback and supplied pill qualification

Published [v0.1.0-rc.5](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.5)
from `b973b5a9cc07197531cfbd1a5e8f60444772e5d2` with
[green source CI](https://github.com/thelifeandtimes/omabit/actions/runs/38793312637).
All eight downloaded GitHub assets passed checksum/provenance/executable/installer
verification. Archive SHA-256:
`445ebde250e128029f6fbc15555a4170baf452285fc92af44c2d399c35dab104`.

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

## Live deployment and RC.6 follow-up

The published RC.5 runner was installed on sourcebox with the owner installer.
Both running ship containers, identities/configurations and both engine processes
were preserved. All 32 observed Urbit processes retained their PIDs/start ticks/
command hashes; account, subordinate IDs, lingering and management access were
preserved. Installed runner SHA-256:
`a0e3ea2fe5c0e5b67b1659429a43c5981e6fc400eb0d8c6ff64a34aad07f449e`.
Private receipt: `/tmp/omabit-rc5-deployment/host-update.json` on sourcebox.

The cold-reloaded RC.5 UI exposed a real integration issue: its raw inline settings
did not contain the manifest's default pill URL. The deployment gate correctly
refused `recommended_pill_default:false`. Omarchy's `BarModel.entrySettings()`
copies explicit entry keys, excluding `id`; the service must resolve defaults.

[RC.6](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.6), source
`2786969d308da9a5b4c948bfc93e3892449657d8`, fixes the service/selector default resolution.
Omitted, custom, legacy and explicitly empty values plus independent bare bootstrap
passed model/mapped tests. A real sourcebox cold reload reported **RC.6** and
**`recommended_pill_default:true`**, with both comets healthy and containers/services
preserved. Receipt: `/tmp/omabit-rc5-deployment/ui-default-fix.json`.

The Go/backend/runtime source tree is identical between the RC.5 and RC.6 build
commits. Sourcebox retains the verified RC.5 host executable implementing the same
backend, while native UI/tool pins advance to RC.6. RC.5 assets were preserved;
the corrected RC.6 archive and tools were published separately and downloaded for
verification. Source CI:
<https://github.com/thelifeandtimes/omabit/actions/runs/38796672732>.

Final native pin synchronization used checkout
`0b64258de69b547b33de1237d583a43acb9dbefa`, followed by another unlocked cold shell
reload. The running UI reported RC.6 and an effective supplied pill default; all
three cached RC.6 executables matched their published hashes/sizes. Both comets
remained running/healthy with the same container IDs and service processes.
Final receipt: `/tmp/omabit-rc5-deployment/ui-final.json`, copied with the host and
default-fix receipts into local `build/rc5-qualification/`. Native-pin CI passed:
<https://github.com/thelifeandtimes/omabit/actions/runs/38797746451>.
The temporary administrator SSH agent was closed after verification.
