# Managed private-gateway session adapter

The real backend now implements the existing internal input/start/observation
interfaces. It uses the installed owner's exact managed root and private engine
boundary. Inputs are selected from `managed/private-runtime-inputs/<manifest-sha256>`;
clients cannot supply runtime paths. Missing delivery remains a refusal. No new
public route, capability, startup dispatch or dependency download is added.

Admission rehashes the sealed generation and fixed initializer, rechecks the held
empty workspace around the engine/package probe, and creates no client metadata
or engine resources. The accepted worker repeats this preflight, then exclusively
creates `private-gateway-client-<operation>/` with an empty Docker config and a
canonical version1 public selection marker. Its files/directories are private and
synced. Partial setup or an existing directory is retained, never repaired/adopted
for another initial attempt. Observation reopens that exact selection read-only.

Held directory/file handles and bounded checks bracket every Docker CLI command.
Replaced objects, aliases, mode changes, marker edits, extra files or injected
Docker configuration refuse. A complete creation ID survives a post-command
failure for durable journalling. Close releases handles and preserves all files;
observation wrappers expose no create/Start methods. These checks assume a trusted
UID and do not provide wallet-safe storage or isolation from same-user programs.

All42 focused race checks/vet pass, including generation07 with genuine temporary
workspace/client directories and inert engine providers. The preflight executes
no CLI command; opening the driver only inspects the pinned image and mounts.
Repeated creation and the obsolete generation06 refuse. The actual qualification
uses an explicitly selected retained bundle; installing it at the production
manifest-named delivery path remains a separate gate. All83 affected queue checks
pass, including replay/update fences, lost replies and cancellation. After
integration, all353 gateway checks pass and the runner test binary builds without
running its CLI. Diff checking is clean. No actual engine or input program ran.

This supersedes M398's missing real backend/client session composition only.
The shared `private-projects` storage parent still requires explicit provisioning;
do not silently recreate a missing parent associated with retained workspaces.
Supported bundle delivery, public lifecycle surfaces, and current-source no-NIC
VM initialization/recovery tests remain. Preparation23 is unbooted historical
infrastructure and cannot qualify the new adapter. Protected ships, tmux ships,
temp-ur and installed plugins/services were not accessed.
