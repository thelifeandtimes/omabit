# Private Lens exit is not a qualified workaround

Both fresh private keyed trials are stopped and their failed piers retained.
Trial01 stops at an overly strict identity comparison: Lens returns `~zod` with
a trailing newline. Trial02 validates the parsed string, confirms the known
SIGTERM-ignore mask, and dispatches `|exit` through private Lens. The request
times out and the process remains alive past the bounded exit wait. Only the
new owned child is force-stopped. Neither trial reaches same-pier reopening. Trial02 initially records its scope as
not inactive immediately after exit; a later exact-unit check confirms inactivity.
The initial retention assertion and the later proof are both retained.

Do not substitute Lens exit for the failed initial keyed SIGTERM contract.
M222's pinned runtime cause remains established. The next experiment builds
a minimal source correction in a separate archive with an explicit experimental
version label, retaining the official binary/image and all installed services.
The candidate must pass an actual signal-behavior regression and a fresh keyed
boot/stop/reopen trial before it advances private devnet qualification.

Secret files and raw bootstrap logs are excluded from the retained public
metadata. These are failures, not a successful native shutdown or devnet pass.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-private-lens-shutdown`.
