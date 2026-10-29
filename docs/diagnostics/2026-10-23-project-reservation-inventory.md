# Read-only registered project inventory

`urbit-collector projects` reads saved project identities, revisions, canonical
review digests, member identities and reserved ports from the selected runner.
`GET /v1/projects` advertises only `project-reservations-v1`. It validates project
bodies and ownership claims in one read-only transaction, returns no internal
container/process/credential data, and explicitly reports runtime observation as
not performed. A saved reservation is not live readiness or launch permission.

Observer access permits exactly this GET route. The client checks capability and
runner/mode binding, canonical plan/digest, deterministic member ordering and
cross-project claims. Older runners are refused before the new route is queried;
existing snapshots remain available. Registration and lifecycle mutations stay
private. No persistent schema change is needed.

Focused corruption, read-preservation, peer/auth/protocol and actual private Unix
RPC tests pass. Full client/access/collector race suites pass in2.084/1.825/7.771s;
vet and four skill-package checks pass. A built collector walkthrough in fresh
namespaces passes private enrollment, JSON/text reads, older capability refusal,
changed identity refusal and offline command help. Its binary hash and all logs
are in the JSON report. Two CLI test-setup failures are retained: invalid pin/socket
combination, then nonprivate preexisting temp directory. Selection and permission
rules were preserved; the fixture now uses a fresh nested registry and enrollment.

This closes the public reservation-read gap only. Running VM23 and unbooted
appliance24 were frozen before this source; neither qualifies the new public view.
No installed runner, ship, desktop plugin or host engine was contacted.
