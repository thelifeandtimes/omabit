# Hosted-style DMs cookie compatibility

A network-free regression reproduced DMs rejecting valid explicit same-host
`Domain` cookies: Python's CookieJar stores these with a leading dot. The corrected
standalone login accepts host-only and exact same-host domain spellings, then
checks the authenticated backend before saving. It rejects ambiguous, parent-domain,
wrong-path, expired or malformed session cookies before saving or starting a channel.
A rejected cookie selection clears the transient runtime account.

The test uses the actual CookieJar parser and restored-cookie request selection,
with inert responses and in-memory credential storage. Before the fix it reports
two errors and two failures; afterward all 33 affected login, bridge, stream,
receipt and storage tests pass in 0.078 seconds. No network, hosting account,
production credential or installed plugin was accessed. Real hosted TLS/gateway
qualification remains open.

Theme and DMs connection documents now identify the already qualified approval
flow and native transport while preserving their dated earlier limits. No shared
client dependency or Omabit data proxy was added.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-hosted-cookie`.
