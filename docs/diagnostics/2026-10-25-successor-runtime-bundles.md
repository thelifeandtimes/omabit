# Complete immutable bundles for linked desktop authorities

Artifact5 adds the explicit successor guard and its complete local Python dependency
set. The generated primary entry still uses SharedHost and the existing ticket1 /
admission2 contracts. New successor-protocol2 declares authority readers1 and2,
reservation readers1/2/3 and journal readers1/2/3. The existing runtime-protocol1 is
also retained, accurately describing the original modules. Neither old metadata nor
old runtime state is relabeled. This changes packaging, not persistent runtime formats.

Preparation verifies separately pinned application/support inputs before staging.
Static inspection checks literal reader declarations and every ordinary Python
import, including imports deferred inside methods, against the bundle or the standard
library. No bundled code executes during preparation/verification. This is a bundle
contract check, not a semantic audit or a security sandbox for publisher code.

The new complete-set preflight requires authority2 by default. It can explicitly
require authority1 for an initial desktop; mixed artifact4/5 passes1 and refuses2.
Historical1/2/3 remain readable but undeclared, and4 retains its authority1 declaration.
The public read-only CLI preserves its authority1 default and adds --authority-format
for an explicit successor check. Existing preflight and writer modules are unchanged.
Refusal and cleanup remain independent of forward admission.

All18 new tests and373 collection tests pass in private HOME/XDG/PID/network labs.
Coverage includes all three roles, actual standalone bundle imports under Python
-I -S without checkout/site packages, static non-execution, wrong declarations,
missing imports/pins, tampering, mixed-version refusal, wrapper binding, lost
publication reply and exact retry. The original artifact4 writer, dependencies,
support and generation were frozen before these additions;43 pinned files verify,
the original writer reproduces its bytes and refuses5 without changing it. Its
current writer also reproduces the frozen fixture. Earlier core/writers are unchanged.

The first draft failed before publication because a deferred startup_recovery import
was absent from its support inventory. It is now included and covered by missing-pin
and isolated-import tests. A later driver selector error accidentally repeated the
focused suite; it is retained and is not counted as a full regression. The corrected
fresh full run passed373 tests. Four scopes are inactive. Exported synthetic packages
and source hashes were independently verified; no private accounts/keyrings/sessions
or production ships were exported or accessed.

M320 supersedes M319's missing compatible immutable-package contract only. Actual
native successor app construction, direct subscription restoration, canonical ready
commit and interruption handling remain the next integration gate. Later session
loss, full-shell/Hyprland, reboot/keyring unlock, public profiles and installed upgrades
remain unqualified. No installed plugin or production runtime changed.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-successor-runtime-bundles
