# Tend compatibility policy

Tend `0.1.0` has one supported deployment shape: install the desk, Omarchy
plugin, and CLI from the same release, and update every ship that shares Tend
lists to that release together. This is a deliberate pre-release lockstep
policy, not a claim that mixed builds are compatible.

## Compatibility matrix

| Desktop | Home desk snapshot | Peer desks | Result |
| --- | --- | --- | --- |
| Protocol 1 | `protocol-version` 1 | Same Tend release | Supported |
| Protocol 1 | Missing or any other version | Any | Refused before the client becomes Online |
| Older or unknown desktop | Protocol 1 | Any | Unsupported; update the desktop |
| Protocol 1 | Protocol 1 | Mixed pre-release Tend builds | Unsupported; update all participating ships together |

The desktop requires an exact `protocol-version` value in every initial and
streamed snapshot. A new login validates that snapshot before it saves either
the Eyre cookie or connection file. Existing connections validate before state
is exposed as reachable and before invitation acceptance or decline. The
Quickshell service validates the first streamed snapshot before changing from
Checking to Online, so an incompatible desk cannot enable editing.

`%tend-peer-1` is also pre-release. Its noun shape may change between builds,
and no rolling-upgrade guarantee exists yet. Before updating, export a portable
owner-only backup and retain a recoverable pier-level snapshot. Update owner
and participant ships as one maintenance operation, then reconnect their
desktops and run the three-ship matrix. If any peer cannot be updated, keep the
whole sharing group on the prior release.

## Versioning rule

Any incompatible change to an Eyre snapshot, action, or update increments the
integer desktop protocol version and must ship with explicit client handling.
Any incompatible peer noun change gets a new mark rather than silently reusing
`%tend-peer-1`. Backward read support and rolling peer negotiation are future
stable-protocol work; they are not release promises for `0.1.0`.
