# DMs profile UI routing

The experimental DMs worktree now derives its overlay ID, visible profile label,
layer namespace and close target from the host-injected manifest. The bar passes
its module ID to the nested popup and reinjects it when the host changes that ID.
Opening the full inbox from a dev/edge popup addresses that instance. Canonical
stable labels and contact-name lookup remain unchanged.

A new networkless Bubblewrap fixture copies only QML/JS into stable/dev/edge roots.
It uses native shell controls with inert process, file and IPC adapters; only the
compositor window is replaced. No Python plugin helper, host home, desktop bus,
compositor socket or production credential enters the namespace. QML warnings
fail the gate even if Qt reports success. Source, fixture and native control hashes
are retained with every final result.

Four behavioral tests (six including Qt setup/cleanup) pass on applied source in
362ms with zero warnings. They cover routes/labels/contact lookup, late manifest
injection, late popup module injection and separate directory-local singleton
state/handlers. The unchanged baseline fails three routing tests and reproduces
an existing unqualified Color namespace ambiguity. The production QML now follows
the native shell's explicit Commons.Color pattern. An early draft also shadowed
an existing displayName contact function; the distinct instanceLabel property
fixes it, and contact lookup is now asserted. All failed drafts and logs remain.

This supersedes only the QML routing portion of M177's pending profile gate.
Native Wayland placement, real IPC, hooks and complete collection switching remain
open. This does not establish full desktop isolation. No installed plugin, desktop
configuration, service or real ship was touched. The concurrent no-NIC public
recovery VM35 is independent of this check.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-profile-ui`.
