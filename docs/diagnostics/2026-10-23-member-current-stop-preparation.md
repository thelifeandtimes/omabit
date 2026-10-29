# Corrected readiness and queued replacement Stop fixture

Appliance21 contains the distinct replacement-readiness request ID from M108 and
a schema34 queued Stop for that replacement. It verifies exact callback order,
retained historical receipts, stopped peer, replay/reopen and clean stopped state.
Ordinary host tests skip and explicit enable refuses the guest marker before
effects. Compilation/vet and164 source hashes pass; the embedded test binary and
guest script independently match their recorded inputs. Verified staging is removed.

Full lifecycle receipts now stay on disk; bounded serial events carry their byte
count and SHA256. Three transport and three framing tests pass, including a1MiB
receipt, malformed/oversized input and failed status. The256KiB event limit is
unchanged. The next offline verifier requires exact extracted bytes/hash before
using the receipt. This addresses M103's growing-event concern without dropping
history. No runtime pass is claimed by preparation.
