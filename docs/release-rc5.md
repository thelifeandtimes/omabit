# Omabit 0.1.0-rc.5 — calmer runtime feedback and supplied pill

- Local setup names the downloaded `omabit`, `urbit-collector` and `urbit-runner`
  executables, their installation locations, OS-supplied dependencies and the
  pinned Vere image. Source/build links cover Omabit, Docker and Vere. Removed the
  unrelated first-start/filesystem paragraph from that pane.
- Recommended Apps defaults to
  `https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill`.
- Refresh logs keeps a stable label, enabled state and layout during polling.
  An inline, fixed-size indicator appears for slower reads. Duplicate refresh
  clicks coalesce into the read already in progress.
- Host port lookup shows progress/errors beside its button, rather than inserting
  a global action banner above the form.
- Validated full public comet names survive log sanitization, including `found
  comet` lines. Runtime details can show that early Vere-reported name while the
  runner continues its authoritative readiness/identity verification. Login-code,
  key and credential-line filtering remains covered by regression tests.
- Fleet cards show **HTTP** and the host port, with a separate **Open →** button.
- Application Connections is a **Coming Soon** stub. Its hidden application-session
  and theme-sync polling is inactive; generic CLI functionality remains available.

Linux/AMD64 prerelease; schema16 and the pinned owner-user runtime are unchanged.
The supplied pill's frozen copy is 59,351,751 bytes, SHA-256
`85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`. It passed a
disposable fake boot/healthy identity+HTTP/clean-stop gate. Detailed application
inventory and OTA behavior are outside that smoke test.
The smoke output also reported Grubbery's "23 files under /code did not compile"
message and HTTP504 retries; these observations are recorded in qualification.

Use the [native update procedure](native-plugin.md#updates), including a cold shell
restart on the unlocked desktop. Host management must also be updated for the log
sanitization change; the runner update preserves running ship containers.
