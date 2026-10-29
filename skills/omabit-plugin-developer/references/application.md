# Application interface and connection lifecycle

## A cohesive Omarchy interface

Use the target Omarchy generation's shared color/font/spacing conventions and
existing plugin controls. Omabit's `plugins/omabit/Design.qml` derives surfaces,
foreground, accent and urgency from `qs.Commons.Color`; inspect it as an example,
not a new runtime dependency for every plugin. Keep one clear primary action,
readable empty/loading/offline/error states and keyboard-accessible controls.
Show the selected ship and experimental profile where the user chooses an action.
Put transport and artifact diagnostics behind details unless they change a decision.

Authentication success does not enable unrelated desktop effects. Theme must not
publish or apply appearance merely because a session was handed over. DMs must not
open a launcher merely because credentials were saved. Let each app's explicit
settings and normal lifecycle control these actions. Distinguish saved settings
from the state actually observed on the ship or desktop.

## Standalone authentication and optional approval

Provide a direct URL/+code route even when an Omabit-assisted route exists. Verify
the intended public ship identity before sending credentials, then verify the
authenticated identity. Bind sessions to a canonical origin and ship. The current
reference clients allow HTTPS or literal loopback HTTP; redirects and ambient
proxies cannot move a secret to another origin. Do not silently relax those rules
to make a connection work.

Keep login codes transient; send them over private stdin or an equivalent bounded
private channel, clear the UI field, and store the resulting session in the chosen
credential adapter. Desktop approval receivers use Secret Service with no automatic
plaintext fallback. Private files and same-user keyring attributes organize state;
they do not sandbox an arbitrary plugin running as that Unix user.

An Omabit approval gives one plugin/profile a separate ordinary Eyre owner session.
It does not grant desk-scoped permissions: an Eyre session can access other desks.
State this plainly in approval. Do not build initial support around Gall 2026
permissions or invent enforcement the current server lacks. Avoid copying one
parent cookie into multiple plugins.

The implemented receiver validates a bounded offer, persists/verifies credentials
and acknowledges the exact attempt binding. Its installed declaration includes
every helper dependency so the selected artifact is stable. Follow the versioned
contract for expiry, unknown fields, durable attempt receipts and reconciliation.
Never treat an uncertain acknowledgement as permission to issue another session.
Cleanup targets only the exact bound account/session; a newer standalone login
and other plugins' sessions must survive.

The plugin's own client sends scries, watches and pokes directly to Eyre. Keep
transport errors, expired authentication, backend absence, permission refusal and
schema incompatibility distinct. Do not repeatedly ask for +code when a backend
is absent. Use bounded reconnect backoff with jitter and restore subscriptions
without repeatedly bootstrapping the entire data set. Talon's
`composeApp/src/commonMain/kotlin/io/nisfeb/talon/urbit/ReconnectPolicy.kt` is a
concrete reference: reconnect and full reconciliation have separate budgets.
Do not blindly replay writes after an uncertain response; reconcile application
state or use a backend-supported idempotency key.

## Backend readiness and installation

For concrete state distinctions and reviewed kiln payloads, read
[Backend setup](backend-setup.md). These are source references; the target
backend still needs a disposable runtime qualification.

After sign-in, probe the exact required backend. For Gall, distinguish a Clay desk
from an installed/running agent and from the API version the client understands.
For Grubbery, distinguish framework, application/nexus, route availability and
required permits. Read the app's source and current installation docs. Lattice's
`README.md` and `docs/grubbery-ops.md` illustrate why an installed framework alone
does not prove an application is ready, and why changing its publisher can affect
other applications.

When a supported backend is missing, present **Install BACKEND from ~PUBLISHER**
inside the plugin. Resolve those identifiers from reviewed plugin-local metadata;
never guess them from a route or display name. Explain the actual desk/nexus change
and show the publisher/source before submission. Preserve existing framework or
desk sources; replacing a publisher or adding broader permits is a separate effect.
The Omart `%pals` installation flow is a reference for named install, progress and
agent-readiness checks. Do not turn this workflow into an Omabit catalog.

Use a verified application install interface and known payload shape. Do not paste
untrusted listing text into dojo/shell commands. Track submitted, installing,
permission-required, ready, failed and unknown outcomes. Resume observation after
restart; do not blindly repeat an uncertain install. If a supported one-click
transport is not implemented yet, label that gap and implement/test it before
claiming the pattern works. A link to manual instructions is not evidence of a
completed one-click gate.

Keep a plugin-local compatibility declaration for backend identity/source, API
version/probe, supported client/backend combinations and artifact/update channel.
Third-party clients must be able to maintain this independently of desk publishers.
Test both update orders and retain a compatible desktop generation when possible.
Do not promise to reverse a ship migration when rolling back a desktop client.
Desk OTA must not silently execute newly delivered desktop code outside the selected
desktop update policy. Ship-served plugin artifacts and Urgit distribution remain
separate experiments until installer support and provenance are demonstrated.
