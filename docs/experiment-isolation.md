# Platform experiment isolation

The user set completion of the platform roadmap as the active goal on
2026-10-21, with the following overriding operational constraints.

## Production exclusions

- Do not operate on the Omabit-managed `~lismyl-lismyl`: no stop, restart,
  migration, runtime replacement, import, copied pier or experimental enrollment.
- Do not inspect or operate on any user's ship running in tmux, or any ship/data
  under `~/temp-ur`, on any host. Do not attach to or manage those tmux sessions.
- Preserve all other identities in [protected ships](protected-ships.md).
- Do not change current installed plugins, their sessions/preferences, the active
  Omabit bootstrap/profile, runner services, Docker daemon or existing containers.
- Prior historical reset/deployment permissions do not authorize any of these
  actions for this goal. Test with freshly created disposable identities only.

## Native manual-review exception — 2026-10-28

The user explicitly replaced VM-only manual testing with separately namespaced
plugins on sourcebox's main desktop and a DEV runner/engine rooted at
`/srv/dev/urbit`. This supersedes the separate-desktop requirement below for this
manual review, including reuse of host dependencies and the real desktop/keyring
APIs with distinct DEV profiles. All production exclusions still apply. See M463
and [the native review guide](native-dev-review.md). Count native DEV storage in
the existing 40 GiB experiment budget. Do not turn human-review state into an
automated test fixture or reset it without authorization.

## Experimental desktop contract

Use separate experimental plugin IDs and visible labels; also isolate every state,
configuration, cache, credential keyring attribute, IPC name, hook and launcher.
Changing only `manifest.json` is insufficient. Keep the normal installed plugin
sources and defaults unchanged. Do not reuse production sessions or enrollment.

Theme can apply Hyprland, theme and font configuration, so its initial experimental
desktop must run with private HOME/XDG roots and a private compositor or inert
desktop adapter. Do not give experiments the production desktop's D-Bus/Hyprland
control sockets or allow their hooks to target the existing plugin. Receiver and
transport tests can run headlessly before any separate UI installation.

Omabit profiles isolate desktop state, but not host runner services or piers.
They cannot by themselves establish this boundary. Collection profiles should
eventually automate the complete supported isolation contract.

## Fresh authentication lab

The initial real-runtime experiment uses Bubblewrap with new user, network,
mount, PID, IPC and UTS namespaces. Mount only read-only OS runtime files, copied
test artifacts, and a newly created private lab directory. Do not mount `/home`,
host `/run`, Docker sockets, SSH agents, keyrings or desktop sockets. Loopback is
private to the namespace; no host port is published and no external route exists.

Copy the standalone Vere executable into the lab and record its hash/version;
this does not update the installed runtime or certify it as Omabit's pinned image.
Download the repository-pinned bootstrap pill and verify its byte count and hash.
Boot a fresh fake `~zod` with no native control socket, no dock, and local-only
network behavior. Apply a deadline and low scheduling priority inside a uniquely
named transient user scope capped at 4 GiB memory, no swap and one CPU core.
This scope contains only the fresh lab; it does not restart or alter an existing
service. LMDB reserves a large virtual mapping, so a small address-space limit
is not a substitute for the actual-memory cgroup limit. Run only tests that verify the namespace guard before connecting.

Retain compact results and bounded logs as evidence. Keep a full pier only when
selected under the retention rule below. Gracefully stop only the child
process created by this invocation; a forced termination, if necessary, is confined
to the fresh process namespace and recorded as a failed clean-shutdown gate. No
global kill commands, Docker cleanup or searches for existing piers are allowed.

Later Docker networking, import, installer and upgrade qualification require a
separate disposable VM/engine boundary. This native authentication lab is not
evidence for those gates. Record the exact evidence and limits at each checkpoint.

## Experiment retention — supersedes indefinite full-copy retention (2026-10-26)

The user paused work because generated evidence consumed excessive disk. M447
and the [development review](development-review-2026-10-26.md) record cleanup.
Do not follow an old bundle's launch instructions when it contains a pruning
tombstone, or assume a historical retained-disk note means that disk still exists.

Limit retained data in `omabit-qualification-evidence` and
`omabit-private-docker-labs` to40 GiB including shared caches. Keep at most three
selected reference/current-failure VM disks by default, with required shared bases
and inputs. Keep one runtime experiment active at a time and declare its additional
scratch budget before launch. Measure storage before/after, reuse immutable
inputs/caches, and retain compact results/source identity instead of duplicating
whole stages. Remove only classified superseded generated data with a removal
manifest; preserve source, original migration fixtures and all production exclusions.
If the next run cannot fit, prune or reduce its scope before proceeding. This is
an operating rule; automated launcher enforcement remains to be implemented.

M448 corrects a cleanup dependency error: always resolve symlinks and qcow backing
chains, inventory their actual targets, and verify all retained dependencies after
pruning. Preserving a `/tmp` alias is insufficient when its target is in an evidence
root. The three pre-review overlays currently lack their shared raw base.
