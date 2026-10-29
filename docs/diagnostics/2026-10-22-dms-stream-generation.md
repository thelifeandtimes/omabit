# DMs stale-stream regression fixes

Three offline tests first reproduced stale-session effects in the experimental
DMs checkout: cleanup after backoff deleted the replacement channel; a queued
acknowledgement followed the current channel instead of its original channel; and
a late contacts read assigned the old book before checking the account generation.

The bridge now binds stream mutations to both account generation and channel,
serializes selection reset with channel mutation, rechecks after backoff, and
checks contact-read generation before assigning state. Old channel facts cannot
pass the listener's selection check while reset changes it. No mutation is replayed.

All 24 focused stream/bridge tests pass. [Source hashes and scope](2026-10-22-dms-stream-generation.json)
identify the isolated checkout. This is offline regression evidence, not a real
ship or complete profile-switching pass. The next real plugin transport gate must
exercise this new artifact; earlier runtime reports still describe their frozen
older artifacts. Installed plugins and their sessions remain untouched.
