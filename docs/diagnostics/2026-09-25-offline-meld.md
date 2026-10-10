# Disposable offline meld acceptance, September 25

The user supplied the readable final report from the root-owned checkpoint 6B
executable running as the dedicated `urbit-runner` account.

- Run: `9b5dfcb57a677f7cf1a2b833e6f65274`.
- Report: `/var/lib/urbit-runner/.local/share/urbit-runner/qualification/9b5dfcb57a677f7cf1a2b833e6f65274/report.json`.
- Profile: `offline-meld`; checkpoint passed; cleanup complete.
- Image/backend ID: `sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd`.
- Every phase passed: engine, image, pill, probe, boot, meld, post-meld, cleanup.
- Live limits observed and pier metadata seen; production runtime qualified false.

Meld started at **2026-09-25T17:27:59.875562582Z** and finished at
**17:28:17.194361497Z**, a measurement interval of **17.32 seconds**. Admission
observed **6.01 GiB available RAM** and **415.15 GiB free disk**; these are rounded
console observations, not resource reservations.

Logical regular-file pier bytes changed from **1,016,840,204** to **571,572,236**:
**445,267,968 bytes (424.64 MiB; 43.8%)** smaller. These are file lengths, not
allocated disk blocks or loom utilization. The interval includes utility lifecycle,
removal, and post-measurement work.

All four containers (`probe`, `boot`, `meld`, `post-meld`) were removed with
`create-uncertain=false`. Both runtime containers recorded private HTTP
reachability. Meld uncertainty was false. The post-meld phase confirms replacement
restart against the newly created pier, followed by clean shutdown.

This is operator-reported live qualification evidence. The agent subsequently
queried the restored management API independently: the same runner identity and
SSH operator were present, alpha was running/healthy at revision 2/2 on port
18081, and panel-check was running/healthy at revision 1/1 on port 18085. The active
runner advertised offline-pack; managed meld requires the next binary upgrade.

This closes the bounded primitive gate. Managed meld/replay/resume is the next
live gate; larger-pier memory policy and destructive interruption/recovery tests
remain separate qualification work.
