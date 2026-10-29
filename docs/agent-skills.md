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
predate skills keep their original asset set. Four packaging fixtures cover exact
bytes/provenance, reproducibility, historical exports, incomplete inventory and
unsafe resources. A future frozen full release build remains a release gate; no
new public artifact has been published by this development work.
