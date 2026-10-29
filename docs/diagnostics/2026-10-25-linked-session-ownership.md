# New desktop ownership from a linked checkpoint

The separate session_continuation writer records a format2 intent from a verified
linked checkpoint2. It checks the complete canonical ancestry, the current immutable
mirror, the unchanged committed journal, the closed plugin cohort and an explicitly
selected new live desktop. Intent and runtime claim publication preserve their
original inode/hash through retries. Missing or changed bound evidence refuses.
No descriptor, guard ticket or application is created by this step.

The `desktop-continuation-*` prefix keeps these intents distinct from the existing
successor intents, linked cleanup and checkpoint records. Original writers remain
unchanged, and the original successor reader refuses format2. A second intent cannot
claim the same predecessor. A changed display, application state, session reference,
unknown write, channel cohort or pending canonical transition prevents claiming.

24 focused/454 full tests pass on the first runs. Unit fixtures retire both prior
runtime directories before continuing, and three actual controller kills preserve
one intent and claim. Wrong or missing ancestor/current checkpoint evidence refuses
before any observation or file publication; synthetic application cohorts are used
only in these unit tests.

The native test uses actual copied Theme/DMs helper locks and a private unlocked
Secret Service. After authority2 apps close and explicit transport cleanup completes,
it captures checkpoint2 and starts a fourth private software compositor. A wrong
cohort refuses before staging. Three controller kills before/after intent publication
and after claim publication preserve the exact new intent and claim plus abandoned
unbound stage. All six sessions, the unknown DM, canonical history and checkpoint
remain unchanged; no HTTP request, new login, message replay or new application
occurs during this ownership step. The fourth compositor is then stopped normally;
no continuation authority is published. The prior runtime-retirement checks still
pass. All34 native injected controller kills pass, four compositor exits are0 and
all three test scopes are inactive.

Independent verification checks source hashes, canonical history and both checkpoint
references, actual cleanup cohort, new intent/claim identities, original records and
immutable artifact hashes. Exported evidence is metadata/code only; no keyrings,
raw sessions/account/application files or piers. Unchanged immutable plugin artifacts
are referenced from M323 evidence. Production resources remain untouched.

This supersedes M325's missing ownership from checkpoint2 only. Application admission
requires the next explicit authority/guard contract and compatible bundles. Loss of a
claimed-but-unpublished successor, pending/unbound cases, public installed profiles,
reboot/keyring and real-backend release gates remain unqualified.
