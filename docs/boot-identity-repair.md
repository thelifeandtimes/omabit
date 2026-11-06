# Correct a mistyped keyed first-boot identity

`urbit-runner repair-boot-identity` is a local, installed-owner recovery command
for an unresolved keyed first boot whose actual identity differs from its
registration. It requires the same ship class, a retained running boot container,
an otherwise idle instance with stopped intent, and explicit identity confirmation.

## Procedure

1. Inspect the selected runner's CLI snapshot and logs. Confirm the intended
   actual identity with the operator, the exact full boot-container ID, and a
   fresh configuration revision. Preserve the existing pier and its location.
2. Stop **only** the owner's `urbit-runner.service`. Keep `urbit-docker.service`
   running. The helper acquires the manager's existing exclusive `.lock` and
   opens only its existing owner-private database.
3. Run as the installed pier owner:

   ```sh
   urbit-runner repair-boot-identity \
     --expected-runner-id RUNNER --instance INSTANCE --revision REVISION \
     --expected-ship '~mistyped-name' --ship '~confirmed-name' \
     --container-id FULL_CONTAINER_ID --request-id UNIQUE_REQUEST \
     --confirm-stop-and-correct
   ```

4. On interruption, retain management downtime and retry the **identical** command.
   Do not edit the journal or clear recovery flags. Intervening manager activity,
   changed bindings or an unexplained runtime/storage replacement require review.
5. After success, start `urbit-runner.service`, inspect a fresh snapshot, and use
   normal `urbit-collector ... operate INSTANCE start --revision NEW_REVISION
   --request-id NEW_REQUEST`. Verify actual identity, healthy runtime and metrics.

The stable instance ID, pier path, ports, bootstrap selection and original receipt
history are retained. The ship identity and a default identity-derived display
label are corrected; custom labels remain intact. Configuration revision advances
once. The original failed operations remain part of the history.

## Evidence and interruption handling

Before shutdown, the helper authenticates the original launch contract, full
container ID, process generation, owned network, mount and pier filesystem
identity. Both public `/~/host` and private Eyre `our` must agree with the confirmed
correction. Only then does it durably journal the evidence and use graceful
`+hood/exit`. The old expected name continues to authenticate launch containment
until removal. No force-kill or new bootstrap occurs.

Clean terminal evidence and removal intent are committed before removal. A lost
removal acknowledgement requires explicit private-daemon HTTP404 for **both** the
full ID and original name; errors and empty filtered listings are insufficient.
The original registration stays quarantined until the final atomic transaction
corrects identity, establishes completed boot provenance and records success.
Successful request replay returns its receipt without runtime effects. Staged key
cleanup remains the normal successful-resume responsibility.

## Support window and verification

This development helper supports the current schema47/pinned-runtime keyed-boot
contract, including the locally installed `0.3.7-dev.20261105.5` trial. It neither
migrates older schemas nor installs a runner update. Its version1 journal is an
opaque `meta` entry preserved by current-schema managers; no runtime field or schema
change is required. A current manager can reopen completed correction and admit
ordinary Start. Pending repair remains protected by existing provisioning recovery;
ordinary manager activity invalidates offline continuation rather than being
silently overwritten. Release publication still requires exact-artifact qualification.

Synthetic tests exercise mismatched keyed boot, same-pier resume, preservation of
data/history/configuration, runner/revision/identity/storage/containment refusals,
duplicate enrollment, clean-stop requirements, interruptions around evidence,
shutdown and removal, daemon absence proof, failed final transaction, replay and
manager reopen. These tests use disposable fixtures, not production ships.
