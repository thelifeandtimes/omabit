# Direct ship-session error guidance

The collector adds optional typed guidance beside its unchanged error/code:
resolve_ship_origin, restore_credential_access and review_ship_login. These
separate identity/redirect refusal, credential-store access failure, and missing
or rejected sessions. Navigation uses local `ships` and offline connection/login
help without runner selection. No action is dispatched automatically. Generic
transport failure and untyped matching message text do not suggest a new login.

The collector race suite, vet and diff checks pass. JSON/text CLI fixtures verify
that missing-session failure makes no HTTP request after enrollment's single
identity probe, and preserves the enrollment. Typed/combined-error fixtures verify
identity and storage precedence. Two initial fixture assumptions were corrected:
registry directory permissions and enrollment's public identity verification.
Neither failure exercised a real ship. [Source hashes/check record](2026-10-22-ship-error-guidance.json).

`ship-status` retains its existing structured status-row behavior. Error codes and
exit statuses remain unchanged. This does not qualify hosted service behavior or
a new release. The endpoint appliance was frozen before this guidance refinement
and does not supply its validation.
