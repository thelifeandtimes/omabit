# Independent admission-root characterization

Two fresh Python processes act only as synthetic shell parents in new private
namespaces. Each holds the actual coordinator lease in its own directory and
issues the same three-member collection specification through its own format1
admission root. All six real admission guards are admitted simultaneously. A
second lease against the first coordinator directory correctly refuses. Closing
rootA leaves rootB open with three live claims; both then close and all six guard
claims end. No QML, helpers, keyring, application traffic or ships are started.

This is an expected characterization of the documented current boundary, not a
cross-root safety pass. Per-root flock and one-use claims cannot be treated as
shared runtime ownership. It makes the M288/M294 remaining gate concrete: a public
profile command must not let changing state-directory paths create independent
authority over the same desktop effects. Source hashes verify and the only scope
is inactive. Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-admission-roots
