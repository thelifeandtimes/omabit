# Private-chain state and Jael survive restart

Fresh trial07 with the experimental signal-fix runtime passes: the native keyed
ship reads the admitted local Azimuth chain, reports initial peer-key life1 in
both its Azimuth endpoint and Jael, observes a genuinely fresh key rotation to
life2, and retains life2 in both after reopening the same pier. All six setup
commands receive Gall acknowledgments. SIGTERM exits0 in892ms and4229ms.

An independent no-network audit reopens a copy of the stopped chain without
sending transactions. Genesis and all contract code match; four receipts and
three ChangedKeys events verify. Head remains0x91, zod life1 and nec life2.
Both stopped pier databases are independently hashed and retained. The primary
213008384-byte database SHA256 is
`f4acdc36935449c17ffb698a1295e07dab5f16990c156f95de0892db3b166429`.

Trials05 and06 remain failed. Both observe initial life1 but mistakenly submit
the same keys again; pinned Azimuth setKeys explicitly returns without changing
revision or emitting an event. Their status1 receipts contain zero logs. Trial05
also keeps the earlier watchdog timer (three chain reads); configuring the one-
second interval before creating the watchdog yields304 reads in06. This solves
the polling setup but cannot turn a no-op into rotation. Trial07 uses fresh keys
and validates the changed chain revision and matching private-key proof before
waiting for ship observation. The30-block lag remains; the fixture mines64
private empty blocks after each actual key change.

The candidate is not installed or released. Two keyed peers, Docker integration,
project isolation/wrong-chain handling and the supported production runtime path
remain. Input secret files and raw keyed bootstrap logs are excluded from copied
metadata. Failed trials, final piers and independent chain audit are retained;
only verified duplicate inactive pier copies are replaced by retained links.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-private-watcher-recovery`.
