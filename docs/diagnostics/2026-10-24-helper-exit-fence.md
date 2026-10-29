# A signal-killed helper owner does not prove descendant cleanup

The actual immutable applicationauth.Artifact receiver adapter is exercised through
a guarded Go qualification owner with the collector's SIGTERM cancellation pattern.
It receives an empty synthetic request; no ship, Eyre session or credential store
is involved. The receiver deliberately starts an inert child in its process group.
Normal completion and context cancellation reap both before the owner returns.
SIGKILL of the owner leaves both receiver and child alive. The private driver
verifies their exact process identities and cleans up only those fixture processes.
Draft and applied scripts/test-receiver-lifecycle.py runs confirm this behavior.
This characterizes the real adapter, not a full collector CLI invocation.

Omabit now records unconfirmed descendant cleanup when any service-owned helper,
application-connection helper or registered reader exits by signal. The transport
receipt includes helperExitsTracked and unconfirmedHelpers, remains closed=false
and reports helper-descendants-unconfirmed. A normal nonzero process exit still
uses that helper's cleanup/recovery contract. There is no automatic flag reset or
claim that production orphan cleanup has been implemented. This is cooperative
lifecycle accounting, not isolation against arbitrary same-user plugins.

The real-QML fixture additionally proves active application-connection work refuses
drain. Signal-killed inventory, runtime-log and application-connection owners all
retain the fence. The inert auth owner leaves a detached child alive after the
shell has no children; the receipt still refuses closure. Exact private cleanup of
that child does not silently clear the fence. Existing five-profile delayed-reader,
late-result, failed-start and lost-owner checks pass, along with the service-only
baseline. Source hashes, host-entry refusal and inactive scopes are independently
checked. All tests use fresh private HOME/XDG/PID/network/mount namespaces and
resource-limited scopes. No production ship, plugin, service or keyring is touched.

M267 supersedes M266's potential clean receipt after a signal-killed owner only.
It does not qualify full collector/SSH/clipboard descendants, a compositor,
automatic recovery from a lost shell or a durable collection transaction. All
receipts retain collectionReady=false. The next checkpoint combines the supported
Omabit/Theme/DMs service set before defining persistent transition/recovery state.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-helper-exit-fence`.
