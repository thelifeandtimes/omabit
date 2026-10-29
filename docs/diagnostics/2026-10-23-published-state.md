# Published Eyre state and subscription recovery — 2026-10-23

Fresh run23-01 passed on pinned Vere4.6 and guest rootless Docker29.9.1. Two fresh
fake galaxies shared an anchor namespace on an ordinary bridge. Eyre traffic went
directly from the guest's loopback publications18281/18282 to public8081/8082;
Omabit did not proxy it. Administrative setup stayed inside the project namespace.
The no-NIC VM had no host filesystem shares or published host endpoints.

The independently checked receipts establish:

- 50 direct public HTTP/SSE requests; four successful peer messages, two local
  controls and one explicit remote rejection.
- Four uniquely acknowledged state writes and13 observations with exact ordered
  contents. The receiver resumed the same pier in a new container; original
  sessions remained valid and no write was replayed.
- The existing subscription recovered its queued update with its saved cursor.
  Subscription revisions were0,1,2,3,3,4 and cursors1,2,3,4,1,2; the final two
  events belong to a new channel reconciled after restart. Both channels were deleted.
- Exact requested/active IPv4 loopback publications before and after replacement;
  two IPv6 connection refusals and two Lens refusals via the bridge address.
  Independent logout revoked both sessions.
- Three clean member stops, clean anchor shutdown(exit143, no OOM), no cleanup
  errors, and clean VM poweroff in1310.48seconds. No deadline/log/quota failure.

All16 extracted receipts, the console, original appliance and disk hashes were
verified independently. Original images and36 metadata files are retained at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-published01`.
Only the verified regenerable raw inspection conversion was removed.
The network receipt SHA256 is
`41f06c700401cb2715d87263ecb91c504f03ec0dac899229702371102c1bc1cd`.

This advances M034 from prepared to passed and extends M033's HTTP fixture to
actual Urbit traffic. It does not qualify simultaneous published projects, actual
ship anchor replacement, runner project ownership, external network security,
hostile members or power-loss recovery. These remain separate gates. No production
ship, tmux session, temp-ur data, installed plugin or host engine was accessed.
