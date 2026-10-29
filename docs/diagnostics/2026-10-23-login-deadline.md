# Confirmed login deadline and source fix — 2026-10-23

Diagnostic run23-04 reached fresh container selection and code retrieval. Its
nonsecret access audit records `stage=identity-after, deadline_expired=true`.
The four-second outer limit expired in the final identity check; the code was
correctly discarded before returning it. This confirms the timeout hypothesis in
M040. It is not an identity mismatch or proof that endpoint discovery succeeded
through the complete gate.

Original artifacts/disks and44 extracted receipts plus66 metadata files were
verified and retained under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-login04`.
The VM powered off cleanly in761.94seconds, with no OOM/quota/log/outer-run deadline
failure. Explicit login/restart completion was not reached.

The successor keeps all admission, exact-container, inventory and before/after
identity checks. The code-read budget is now12seconds, with15seconds for this
response/client only. Ordinary RPCs keep their five-second limit. Three-second
command and two-second curl limits remain; no automatic retry was introduced.
The server extends this response's write deadline, and the client copies its
transport configuration so other requests retain their ordinary timeout.

Race tests cover existing refusal/redaction cases and a real private Unix HTTP
connection with ordinary five-second server/client limits and a5.5-second backend.
That response succeeds; a short caller deadline interrupts the backend promptly
and sends no duplicate request. Client-default tests,10 fixture checks and vet pass.
The actual runtime fix still requires a fresh VM gate; these are source checks,
not an installed or real-ship success claim.
