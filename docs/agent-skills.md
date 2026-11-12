# Agent skills shipped with Omabit

Source checkouts include `skills/omabit-operator/` and
`skills/omabit-plugin-developer/`. Each is a complete skill directory: keep its
`SKILL.md` and `references/` together. Agents can read the relevant SKILL.md directly
from the checkout; an agent-specific global installation is optional and explicit.

The binary bundle and flattened plugin export include both skills under `skills/`
and the CLI reference under `docs/cli/`, covered by the bundle checksums. The native
Git checkout carries the same source guidance.

The packagers also produce a separate `omabit-skills-VERSION.tar.gz`
asset alongside the binaries and desktop bundle. The release checksum list and
asset metadata cover the archive. Inside, `SKILLS.json` records the source commit,
version and every resource hash; development snapshots also retain their development
label/source binding. The archive's own `SHA256SUMS` covers the complete payload.
The two skill directories remain under its `skills/` directory. No install hook,
executable resource or automatic agent-configuration change is included.

Use instructions matching the intended binary generation. Verify the archive
against that release's metadata/checksums, extract into a fresh directory, then
read the selected skill or explicitly import that whole directory into an agent's
skill system. An existing custom skill must be reviewed before replacement. The
skill still requires `hello` and command help to establish actual capabilities;
its presence does not upgrade an older runner or authorize a fleet action.

The normal desktop loader continues selecting its existing UI/tool bundle; it does
not activate skills or modify the agent's settings. Historical source exports that
predate skills keep their original asset set. Five packaging fixtures cover exact
bytes/provenance, reproducibility, historical exports, incomplete inventory and
unsafe resources. The0.3.0 frozen release includes both skills and the complete
CLI reference; see [qualification](releases/0.3.0/qualification.json) for exact
source and artifact identities. Publication remains subject to protected PR checks.


## Agent Help and skill installation in 0.4.0 development

The monitor's **Agent Help** button uses `omarchy default agent` and
`omarchy agent prompt`, following Omarchy's crash-diagnosis flow. A failed or
interrupted remote setup also has a context-specific button. The prompt points
at the exact activated bundle's operator skill and a quoted, read-only command
for the selected profile and attempt. It excludes raw SSH output, credentials
and host-supplied free text. Help requests diagnosis, not repairs or exposure.
The selected agent still owns its sign-in, execution settings and terminal.
The interactive launcher runs in its own process session, with no inherited
Omabit input/output pipes. Omabit checks immediate startup failure for half a
second, then acknowledges the launch without waiting for the terminal to close.
The helper deadline and panel shutdown must never end the user's agent session.

**Setup → Agent Help and skills** shows the default agent and both bundled
skills. If no default exists it points to Omarchy's **Setup → Defaults → Agent**.
The feature does not select, install or configure an agent automatically.
Agent Help works without a global skill installation because the prompt includes
the exact skill path. Older bundles without verified skills report that they
need a matching updated bundle instead of falling back to another release.

The installer proposes OpenCode's global skill directory, currently
`~/.config/opencode/skills` (respecting XDG_CONFIG_HOME), as documented by
[OpenCode](https://opencode.ai/docs/skills/). For other agents, the user enters
that agent's documented directory and reviews it. Automatic discovery paths for
other harnesses are not asserted by this implementation. Agent launching still
works through Omarchy for whichever supported agent is selected.

Each explicit installation copies the complete skill and its references. The
review binds the selected source and destination state. Existing custom skills,
links, extra resources and edited managed copies are preserved; an update is
allowed only when the previous Omabit receipt matches every current file.
Updates exchange directories atomically and retain the previous copy outside
the skill-discovery directory, under the sibling `.skills-omabit-history` (or
corresponding name for a custom target). A changed review must be refreshed.
No background synchronization or automatic overwrite is performed.

New binary bundles and development snapshots include SKILLS.json alongside the
complete skills. Native bootstrap retains them and verifies their bytes against
the authenticated archive, not just the mutable cache inventory. Historical
bundles without this declaration retain their old cache layout. Flattened plugin
exports retain the same build/skill provenance. These are packaging and feature
checks; stable release qualification remains deferred until the other 0.4.0
features are ready.

Validation uses fake agents and private installation directories: helper tests
cover no/default agent, single-argument prompt dispatch, context rejection,
complete installation, edited/custom preservation, stale review, atomic updates,
source mismatch and two skill generations. A networkless full-panel fixture
exercises both installations, launch, no-default guidance and failed-setup help.
The `--persistent-agent` variant keeps two inert sessions alive beyond21seconds
(past the old15-second launcher and20-second QML helper deadlines), verifies
separate process sessions/private stdio, and suspends/closes the Omabit panel.
No real agent was launched or user skill directory modified for these tests.
