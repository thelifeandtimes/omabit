# Pinned keyed-boot signal failure isolated

A fresh fake under the same Node launcher reaches identity and exits0 on SIGTERM
in63.191s total, without force. Its SIGTERM handler is caught, not ignored.
A separate fresh keyed bootstrap is traced only until immediately before worker
creation. It exits the debugger intentionally there; no ship shutdown is claimed.
Both private4GiB/no-swap/one-core scopes are independently confirmed inactive.

At pinned Vere5a7614f, `_boothack_key` calls `u3m_soft`. Its successful
`u3m_soft_top` path calls `_cm_signal_done` at manage.c1536, restoring the libuv
handler, then calls it again at1550. The second call installs SIG_IGN;
`_king_sign_move` runs again without another libuv handler installation. The
pre-worker mask independently confirms SIGTERM ignored. This explains M219's
initial keyed-boot failure and narrows the fault away from Node inheritance.
No runtime source or deployed binary has been changed. A production fix still
requires a qualified runtime change; a diagnostic result is not a shipping fix.

The next private trial tests an identity-checked Lens `|exit`, then reopens the
same disposable pier and checks normal SIGTERM. That is a separate workaround
qualification, not permission to hide the initial signal failure. Full private
chain observation and peer networking remain open.

Raw keyed bootstrap logs and all fresh secret files stay in the original private
lab. Retained debugger evidence includes only signal masks and stack frames with
arguments suppressed; no key values enter these notes or public test artifacts.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-keyed-signal-diagnosis`.
