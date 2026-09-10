# Omarchy Urbit: chat-to-repository handoff

Prepared September 10, 2026 for continued local development.

## What this bundle is

This is the existing `0.1.0-pilot` source plus contributor instructions and this
handoff. It is not a new application release or a finished Omarchy plugin. No
Go, runtime-script, installer, or systemd implementation was changed for this
handoff. Prebuilt executables were removed to make local development use source.
The README packaging wording was adjusted, common credential/state paths were
added to .gitignore, and SHA256SUMS now covers
the source bundle rather than the removed release executables.

No repository has been created in the user's account, no Git remote has been
configured, and nothing has been installed on the user's machines. The original
chat archive remains a separate artifact. This handoff is a curated continuation
record, not a verbatim transcript.

## Purpose and authoritative context

Build an Omarchy integration for managing Docker-hosted Urbit ships. The typical
user owns an always-on workstation and a laptop, connected over Tailscale and
SSH. Ships default to the workstation; the laptop provides private browser,
clipboard, Dojo, CLI, and eventually bar/pane access. One Linux user owns each
host's fleet; a persistent user service is acceptable.

Start with the files below, not a reconstruction of the chat:

| File | Purpose |
| --- | --- |
| `../AGENTS.md` | Contributor workflow, first task, and operational boundaries. |
| `DECISIONS.md` | Settled product direction and expressly deferred questions. |
| `STATUS.md` | Implemented vs simulated-tested vs not live-validated vs missing. |
| `PILOT.md` | Disposable workstation/laptop acceptance test and commands. |
| `SECURITY.md` | Owner/agent authority, credential limits, and data protection. |
| `DEVNET.md` | Same-host fake groups and proposed cross-tailnet work. |
| `SOURCES.md` | Upstream references from the initial investigation; verify current code when needed. |
| `../skills/` | Agent/operator instructions for using the runner, not developing it. |

Key product boundaries already settled: separate real and fake ships; multiple
isolated fake groups; eventually fake galaxies communicating across the tailnet;
restricted real-ship agent access; fleet-budget warnings rather than hard caps;
archive before separately confirmed purge; and no backups in scope.

The planned Omarchy UI is one fleet icon with host-grouped rows, colored
container/runtime icons, and one enabled/disabled GUI button rather than a
separate browser-status icon. Basic pane metrics are identity, CPU/memory, and
endpoint health. Other/heavy details are on demand. Fetch login codes only for
running ships, copy on the client, and open the ship home interface. GroundSeg's
existing-container tmux Dojo is the reference.

Hosted-Eyre access without SSH is deferred. SSH-accessible VPSes use the ordinary
runner. Anchor/StarTram is a later exposure-provider project, initially only the
user's machines on `rooftop`; the user handles Native Planet outreach separately.
Source editing, desk mounts, and other development conveniences remain open for
later discussion, not assumptions for the current pilot.

## Current engineering reality

There is working source for a CLI, user daemon, Docker command generation,
persistent operations, SSH/Unix-socket transport, permission split, private
access helpers, and same-host fake-group isolation. The registry is atomic JSON,
not SQLite. The current runtime readiness field is HTTP-derived, not an
independent kernel probe. Budget warnings are currently per host. These are
implementation facts, not claims that the complete product is finished.

The previous test report used a simulated Docker executor. During this handoff,
`make check`, `make build`, and `./urbitctl version` were run successfully again
with Go 1.23.2 on Linux amd64. This did not exercise a real Docker daemon, Vere
boot, Tailscale connection, Omarchy shell, clipboard, or remote host. No live
integration has been demonstrated in this chat.

Cross-tailnet fake networking, the Omarchy UI, desktop notification delivery,
deep metrics, runtime replacement, and hosted-Eyre/Anchor integration remain
unimplemented. The present fake-group topology is same-host only. Read STATUS.md
for the complete boundary.

## Establish the repository

Download `omarchy-urbit-project-starter.tar.gz` to the machine where development
will run. Use a fresh directory, not an existing checkout or pier data directory.
The example uses `~/Projects` and assumes the archive is in `~/Downloads`:

```sh
mkdir -p ~/Projects
# Check that ~/Projects/omarchy-urbit does not already exist before extracting.
tar -xzf ~/Downloads/omarchy-urbit-project-starter.tar.gz -C ~/Projects
cd ~/Projects/omarchy-urbit
sha256sum -c SHA256SUMS

git init -b main
git add .
git diff --cached --stat
git commit -m "Import Omarchy Urbit pilot and project handoff"
git switch -c pilot/live-validation

make check
make build
./urbitctl version
```

This requires Git, make, Go 1.23 or newer, and a native C compiler for race tests.
It does not require Docker for the included automated checks. Configure Git's
local user.name and user.email when needed; choose your own identity rather than
copying somebody else's author details. The checksum file verifies the supplied
snapshot, not the future edited working tree; it is not a signed release.

The development source belongs under `~/Projects/omarchy-urbit`. Ship data belongs
under the runner's XDG data root, normally `~/.local/share/omarchy-urbit`, or an
explicitly configured dedicated root. Do not put piers inside the source checkout
or synchronize live piers through Git or a file-sync service.

A Git remote is optional for the first local build. Before publication, review
for credentials, private identifiers, and upstream-derived code/license
obligations. This bundle does not choose a project license. Keep the initial
remote private until ownership and the desired release license are resolved.
Use an empty remote and the commands that your Git host supplies.

## Continue with a coding agent

Run your coding agent from the repository root. For OpenCode:

```sh
cd ~/Projects/omarchy-urbit
opencode
```

The root AGENTS.md is the entry point. It deliberately instructs the agent to
read the relevant documents; do not assume a Markdown link automatically puts
all linked content into model context. No separate chat-memory import is needed.

Suggested first message:

```text
Continue this existing Omarchy Urbit project. Read AGENTS.md, docs/HANDOFF.md,
docs/DECISIONS.md, docs/STATUS.md, docs/SECURITY.md, and docs/PILOT.md first.
The source is authoritative for current behavior; the decisions document is
product direction, not proof of implementation.

First reproduce make check and make build, then inspect the runtime-image
adapter, installer, and remote pilot for concrete blockers. Fix confirmed
problems with regression tests. Do not start by rewriting the architecture or
building the UI. Keep the real-ship gate off.

The next milestone is one disposable fake ~zod running in Docker on the
workstation and managed from the laptop over SSH/Tailscale: inspect, open GUI,
copy code locally, attach/detach Dojo, disconnect/reconnect, stop/start the same
pier, and archive while retaining the pier. Read docs/PILOT.md for details.

Before live changes, ask me only for missing environment details and approval
of the specific disposable target. Never request a real networking key or +code.
Do not install services, boot containers, change Docker/SSH/Tailscale privileges,
purge anything, or deploy on rooftop without that explicit target authorization.
Record actual evidence and remaining blockers in docs/STATUS.md and HANDOFF.md.
```

The above instructions guide an agent; they are not a security sandbox. Do not
give an untrusted agent unrestricted workstation/Docker credentials and expect
repository text to constrain its capabilities. See SECURITY.md.

## Next tasks, in order

### P0 — audit and reproduce locally

Build and run tests. Inspect the installation path, runtime image dependencies,
UID/GID handling, embedded scripts, console attachment, and current test coverage.
Add regression tests for specific defects; do not claim live acceptance here.

### P1 — one actual remote disposable ship

Obtain the workstation SSH alias and verify the intended account/data root.
Follow PILOT.md. Record source commit, image digest/runtime, container/runtime
observations, and redacted results. Container-action success alone does not
prove a usable ship. Do not use a valuable existing pier.

### P2 — lifecycle and same-host network acceptance

Check stop/start, laptop disconnection, service restart, workstation reboot
ordering, archive retention, and same-group fake messaging. Use a second group
with another fake ~zod to test isolation. Do not purge by default.

### P3 — subsequent work, after runner evidence

Add the Omarchy fleet UI and independently scoped cross-tailnet fake-group
transport. Preserve both requirements; do not describe existing SSH management
as cross-host fake-to-fake networking. Deep metrics and public exposure come
later. Source-editing/mounting questions remain open.

## Development/deployment distinction

`make build` writes `./urbitctl`. It does not install or restart the user daemon.
The original release installer uses prebuilt files from `dist/` when present;
this source-only bundle omits them so installation builds the current checkout.

Reinstalling a binary does not replace the executable already running in the
host service. Restart that service deliberately between operations after
installation when required. Ship containers are independent of the manager's
process lifetime; changing/restarting the manager does not automatically
recreate their runtime or replace their image.

Embedded runtime scripts are immutable at a particular Version directory. A
script change needs a version-aware deployment plan or a clean disposable root;
never force-overwrite scripts that an existing container is using. Follow
AGENTS.md and inspect the actual InstallScripts behavior.

Keep development and operational permissions separate. Rebuilding source,
installing a service, restarting a manager, stopping a ship, and purging a pier
are different actions with different consequences.

## Maintain context after leaving the chat

Commit source, decisions, tests, and useful redacted acceptance reports together.
Keep STATUS.md honest and update this file's next steps when a milestone passes.
A future agent should be able to continue from the repository, without the full
chat transcript. This chat can remain a planning reference rather than a hidden
dependency of the implementation.

## Tool documentation checked for this handoff

OpenCode's current rules documentation covers project-root AGENTS.md and the
need to load referenced files explicitly:
https://opencode.ai/docs/rules/

OpenCode's CLI documentation covers starting the TUI in a project directory:
https://opencode.ai/docs/cli/
