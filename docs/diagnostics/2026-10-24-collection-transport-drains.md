# Repeatable collection transport prerequisites

The repository lifecycle fixture now has an explicit --connected --drain mode.
It preserves one unknown DM through stable→dev→stable→edge→stable, deletes all ten
Theme/DMs channels before replacement, retains Theme state and never resends.
The original connected baseline still passes with zero application writes. The
synthetic server permits the single missing-receipt poke only in the explicit
drain mode. Entry guards reject host execution and drain without connected mode.
This applies M255's draft fixture as reproducible source and adds edge coverage.

Omabit's actual application Service now offers an internal stopTransport method.
It refuses during foreground/setup/clipboard/first-boot work, pauses admission,
stops inventory reconnects and tracks the inventory exit event separately from
the requested Process.running state. Other service-owned reads finish naturally.
Its receipt stays closed=false while any owned helper/read is outstanding.
The new test runs actual Service/ApplicationConnections/ImportRestoration code,
inert exact-command tools and a monitor stub in a guarded private namespace.
Five profile cycles pass. A delayed SIGTERM exit and an intentionally late runner
read cannot produce early closure or restart the observer; new mutation and auth
requests remain refused. Exact process identities are checked before replacement.

This does not modify bootstrap activation behavior, prove real collector child
transport shutdown, cover monitor-pane processes or authorize a full collection
switch. All receipts retain collectionReady=false. Omabit's normal runner/pier
operations are never invoked by these fixtures. Existing installed services,
plugins, profiles, ships, desktop and keyrings remain excluded.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-transport-drains`.
