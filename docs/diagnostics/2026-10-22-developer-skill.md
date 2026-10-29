# Initial plugin-development skill

The source now includes
[omabit-plugin-developer](../../skills/omabit-plugin-developer/SKILL.md), with
focused interface/authentication/backend and development-isolation references.
It anchors the work in **“Multiplayer apps without a subscription.”**

The guidance keeps plugins standalone, including hosted-ship URL/+code sign-in.
Omabit-assisted approval hands off an independent ordinary Eyre session; all
application traffic remains direct to the ship. It describes the current lack
of desk isolation and leaves Gall 2026 permissions out of initial requirements.
Theme and DMs remain the first integration pair. Backend installation belongs in
the app that needs it, with an exact reviewed publisher, readiness/permit states
and compatibility policy; catalog discovery remains in Omart.

Development guidance covers separate IDs/labels, credentials, state, IPC, hooks,
launchers and desktop effects. It distinguishes current profile mechanisms from
unimplemented stable/dev/edge collection switching, and keeps private local
Ethereum chains separate from public testnets. Wallet work remains research.

The references were checked against Omabit's current design/handoff documents,
Talon's reconnect policy, Lattice's README and Grubbery operations notes, and
Omart's named `%pals` setup component. These are reference patterns, not newly
claimed runtime APIs. Skill structure/reference validation and whitespace checks
passed; [source identities](2026-10-22-developer-skill.json) are recorded.

This is source guidance. It has not been installed into the user's global agent
or plugin directories, included in a published release, or qualified by building a
new application from it. One-click backend installation, collection switching and
the remaining live application gates are still implementation/acceptance work.
