# Tend security model

Tend removes a centralized application identity provider and database, but it
does not remove trust boundaries. This document records the boundaries that a
release and its reviewers must verify.

## Desktop and Eyre

- Users enter a ship's base URL and a current `+code`. The code is sent only to
  that endpoint's `/~/login` route and is never written to disk or placed in a
  process argument.
- Non-loopback endpoints require HTTPS with the host operating system's normal
  certificate validation. URLs containing credentials, paths, queries, or
  fragments are rejected.
- Tend retains only the resulting Eyre cookie and non-secret endpoint/ship
  metadata. Credential directories are mode 0700, files are atomically written
  mode 0600, and symbolic-link credential paths are refused.
- Disconnecting removes the local cookie and endpoint record. Expired sessions
  stop automatic reconnects and require a fresh `+code`.
- Authenticated requests do not follow HTTP redirects. A new login starts a
  fresh cookie jar, so connecting directly to another ship cannot retain a
  usable session for the previous endpoint.
- JSON responses and SSE events are capped at 32 MiB; outbound action JSON is
  capped at 1 MiB. Invalid UTF-8, malformed JSON, malformed event IDs, and
  oversized data terminate the bridge with a structured error.
- Date/time editor values are converted to canonical Urbit instants in the
  bridge with system `zoneinfo`. The bridge strips its private all-day-minute
  hint before measuring or sending the action; Gall continues to validate the
  resulting schedule noun and never trusts display-only local-time fields.
- The Omarchy plugin executes unsandboxed as part of `omarchy-shell`, like every
  third-party shell plugin. Users should inspect the release before enabling it.

## Gall authority

- Gall state on the user's ship is authoritative; the desktop cache and UI are
  not databases.
- Eyre actions are accepted only when Gall reports `src.bowl` as the local ship.
- Inter-ship authority must be derived from the authenticated Gall source ship,
  never from an `author` or ship string inside a payload.
- The list owner is the sole sequencer. Participants can read their last
  confirmed replica while the owner is unavailable, but cannot submit or queue
  shared mutations until catch-up has established Online status.
- Membership removal must revoke active subscriptions and subsequent mutations.
  Operation IDs are deduplicated so network retries cannot apply a mutation
  twice.
- Reminder assignment is authorized from canonical ownership/membership state;
  a syntactically valid ship that is not a participant is rejected.

## Shared-data disclosure

Inviting a ship authorizes disclosure of the selected list's title, sections,
reminder titles and notes, URLs, tags, schedules, completion state, assignments,
and membership metadata to that ship. The owner UI must present the target ship
and disclosure boundary before sending an invitation. Attachments are not in
the current release scope.

## Input and resource limits

- Gall marks validate every incoming shape before agent logic runs.
- Titles (1 KiB), notes (64 KiB), URLs (8 KiB), appearance/tag/time-zone
  strings (128 bytes), recurrence/offset sets, preference arrays, lists,
  sections, reminders, and batch selections are bounded server-side. URL
  actions accept only HTTP, HTTPS, and mailto data.
- `%tend-peer-1` is noun-only and versioned separately from Eyre JSON marks.
- Release installers reject broad desk targets, refuse symbolic-link targets,
  and do not overwrite existing installs unless `--force` is supplied. Forced
  replacements are moved to timestamped backups.
- JSON exports contain complete owner-authoritative reminder data. They exclude
  authentication material, are written atomically with mode 0600, refuse
  symbolic-link targets, and require `--force` before replacing an existing
  regular file. Offline validation opens only a regular file without following
  a final symbolic link, caps it at 32 MiB, and validates nested references and
  cycles before restore can use it. Export filters visible replicas from
  authenticated access metadata. Restore requires explicit confirmation,
  accepts only empty Gall state, commits atomically, verifies a durable receipt,
  and resets all ACL/subscription/transient peer state.

## Release gates

A multiplayer release is not complete until tests prove authorization,
revocation, owner-offline read-only behavior, idempotent retry, migration from
every persisted state version, and operation across planet, moon, and comet
identities. `make check-tend` and `make check-tend-release` cover the local
plugin, transport, CLI, and package surfaces; live multi-ship tests cover the
Gall boundary.

The operation-receipt map retains the newest 4,096 results. Peer operations are
idempotent only while their receipt remains in that window; clients reject
offline writes and bound mutation age so an old request is not silently treated
as a safe retry. The live three-ship evidence is recorded in
`TEND_MULTISHIP_MATRIX.md`; backpressure, mixed-version, and longer soak cases
remain release-candidate gates.
