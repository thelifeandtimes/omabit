# Real private collection under an external supervisor

The optional --collection-supervisor mode runs the real private QML shell as the
sole workload of an external subreaper. The supervisor starts before QML and its
owner nonce/PID/start/boot/namespace is recorded alongside each loaded activation.
A separate adapter controls that guardian and accepts closure only for its original
identity, root process, stop request and durable ECHILD receipt. A replacement
shell receives a new guardian and cannot supply an old owner's cleanup proof.
The normal unsupervised test mode and production coordinator journal are unchanged.

The complete private stable→dev→stable, partial-edge rollback and six coordinator
kill cases pass. Four successive shell guardians produce independently verified
closure receipts. After deliberate SIGKILL of the third shell, the original
supervisor reaps its two surviving inert Omabit readers; the old activation's
saved supervisor binding matches that exact receipt. The new shell remains empty.
All16 Theme/DMs channels close and uncertainty/application records remain retained.
The full four-case orphan/late-fork/EOF/guardian-loss regression also passes using
the final supervisor source. Both scopes are inactive; source/receipt hashes verify.

The collection journal deliberately remains format1 and pending drain-previous
with reader-owner-lost. M280 supplies independently bound process closure for this
supervised fixture, not a mechanism to erase that fence. Next freeze the exact old
activation's transport/application cohort, verify plugin-owned cleanup without app
replay or credential removal, and add a versioned explicit recovery transaction.
Ordinary new subscriptions, a replacement guardian or edited receipt JSON cannot
substitute for bound observations. Guardian loss must still refuse recovery.

This uses real plugin code but private identity instrumentation, inert Omabit
collector/desktop commands and synthetic Eyre. It is not native loader rollout,
full external desktop-service accounting or a production switch command. Protected
ships, tmux/temp-ur, installed plugins and credentials were never accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-supervised-collection`.
