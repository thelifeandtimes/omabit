# Enrollment and key setup refinement — panel 0.17

## Enrollment bug

The live catalogue confirmed import-check and panel-check were already enrolled.
Their false `moved` fields were omitted from JSON. QML's expression
`!enrolled_by || moved` consequently returned undefined, leaving the default-visible
action exposed. The service correctly rejected duplicate enrollment, but returned
false without feedback. Boolean checks are now explicit, ineligible actions are
disabled/hidden, and rejected dispatch has an actionable message.

Each review now expands inside its source card. Draft fields and confirmation live
outside the replacing inventory delegates, preserving edits across polling. Fixtures
include enrolled records with an omitted moved flag and exercise actual card actions.

## Desktop changes

- Shared fresh-selection CopyCodeButton for both menu and full window.
- Short, theme-matched clipboard toast: “Copied to sensitive clipboard for 30s”.
- Activity @p text reduced from 19 to 13 logical pixels; Task Records label.
- Circled, optically centered tilde menubar icon.
- Owned key setup needs only expected @p and private key file. Instance ID and tier
  path are derived; advanced options contain Ames/HTTP ports and URL/local boot pill.
- A blank `defaultBootPill` plugin setting is ready for the future Omarchy address;
  blank effective selection uses Vere's canonical default, not the development pin.

## Runner/collector changes

Port allocation is transaction-bound and idempotent, considers host listeners and
both desired/applied reservations, and keeps TCP/UDP namespaces separate. New owned
ships publish selected Ames UDP; HTTP stays loopback-only. Existing registrations
retain their prior networking and pill contracts.

Custom pills are bounded at 512 MiB, staged/hashed by the desktop collector, streamed
over the identity-pinned authenticated connection, and verified by the runner. The
bridge grants bulk framing only to the dedicated upload route and preserves the
small-body limit elsewhere. The runtime copies/verifies the selected artifact,
while canonical bootstrap omits both pill-file mount and override flags. Schema 10
prevents older runners from ignoring these options.

## Verification

Full Go tests/vet, targeted race tests, native/ARM64 builds, panel model/installation
and mapped-window QML checks passed. Tests cover inline enrollment, omitted booleans,
menu/full-panel code parity, short toast text, inferred paths, automatic port replay,
canonical/custom runtime flags, changed UDP-binding refusal, pill URL/file staging,
hash/size checks, operator-only uploads and streamed SSH bodies beyond the normal
RPC limit.

## Live verification

The runner and panel are deployed; IPC confirms panel 0.17.0 loaded on Setup.
Build, host-installed and panel-bundled runner hashes match:
`0946e38f0ab61566d9b42cf1d27d44ee9c6bc7f09fe1fc04644525a96fc322fb`.

`scripts/validate-boot-pill.go` transferred 71,680 inert fixture bytes through the
actual identity-pinned SSH connection and updated bridge. The runner acknowledged
SHA-256 `aeab266310969e0950576f6e581d28127115fcff0921807630e462c5d8ba249c`.
No registration or runtime start was requested; the artifact is an inert transport
fixture in the private cache, not a boot-qualified pill.

Final readback confirmed seven registrations, all five initialized piers correctly
enrolled, alpha/bus-lab/comet-check running healthy, import-check/panel-check stopped,
and rus/ssh-check uninitialized. Saved policies remain disabled. There are no active
operations, recovery requirements or incidents. No live owned-key bootstrap or new
Ames listener was created for this update; those runtime options have fixture checks.
