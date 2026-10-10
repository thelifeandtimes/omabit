# Start from owned-ship registration — panel 0.18.2

The owned-key form now displays Start alongside Register ship. It starts disabled
and enables when the selected runner's current inventory contains the matching
non-fake instance and Urbit identity in a startable state. Busy, unavailable,
observer, recovery and active-operation checks reuse the existing lifecycle guards.
The action submits the current configuration revision and opens the ship's overview
to follow boot progress. Registration itself never requests Start.

The form also recognizes an existing registration after a panel reload: re-enter
the Urbit ID to restore Start without supplying the key again. Register ship stays
disabled for that identity, and Register another ship clears the form.

## Verification

- Mapped QML fixtures cover disabled-before-registration, disabled-during-submit,
  enabled-after-inventory, reset-to-disabled, different identity/runner rejection,
  restoration without a transient receipt/key, and explicit Start with one request.
- The fixture verifies the selected runner/instance/configuration revision and
  overview navigation; Start disables while dispatching and once running.
- Existing native-dialog regression, model and installer checks pass.
- Captures in `/tmp/opencode/omabit-key-start/` were reviewed, including the
  enabled Start action in the narrow registered form.
- `git diff --check` passes.

## Deployment

Panel 0.18.2 is installed and its loaded version confirmed by monitor IPC.
Source/installed `KeyBootPane.qml` SHA256:
`e8fbbeea46c74b02726050df5a5c900815bb10f3b8ba3662b411bcc362c6c260`.
The previous plugin snapshot is backed up at
`/tmp/opencode/omabit-panel-before-key-start.tar`.

The live fleet now contains eight registrations, including the user's registered
moon `~dibpen-fogweg-dozzod-sarlev`. After this panel update it remains uninitialized,
at configuration/applied revision 1/0, awaiting the user's first explicit Start.
The other seven instances retain their preceding running/stopped/uninitialized
states; the three running ships are healthy. No active operations or incidents
were reported. The regression Start request ran only against the isolated fixture.
