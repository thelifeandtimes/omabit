# Collection recovery across private shell replacement

The optional --collection-shell-recovery mode extends --collection-journal with
actual private Quickshell replacement. After all supported plugins drain and
unload, the shell is cleanly replaced with an empty shell between checkpoint
effect and receipt. It is replaced again after verified offline restoration and
before its receipt. Exact new shell identities are recorded by the fixture; fresh
coordinator processes resume the same durable transitions. Stable/dev/stable and
partial edge rollback still pass, with all six coordinator SIGKILL cases.

A separate final switch intent is then recorded while stable is active. SIGKILL
of that exact private shell leaves two recorded helpers alive and one DMs channel
undeleted. The adapter observes loss of its bound owner and persists reader-owner-lost.
A fresh empty shell cannot advance the request. The fixture terminates only its
recorded surviving PID/start identities and verifies their exit; that independent
cleanup still does not clear the journal fence or enable the candidate. The final
state remains pending drain-previous with explicit uncertainty. The last shell
exits cleanly and the enclosing scope is inactive. An empty new shell or closed
socket is therefore not proof of completed plugin/remote cleanup.

M271 supersedes M270's missing shell-replacement test for this private loader
fixture. It does not implement production orphan/channel recovery, a stock loader
identity API, full native shell/compositor or native font-triggered replacement.
The qualified recovery still uses private source/instance instrumentation, inert
Omabit/desktop controls and synthetic Eyre. The unchanged store/driver already
have M270's30 checks; this new runtime mode exercises actual owner death. Source
hashes, retained journal replay and exact scope exit were independently verified.
No installed plugin, production service, credential or ship was changed.

The next recovery contract needs plugin-owned durable channel/operation evidence
and independently verified cleanup before an explicit fence resolution. Reload
must never become an implicit reset. Keep direct plugin-to-Eyre data flow and
standalone plugin operation; Omabit must not become their data proxy.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-shell-recovery`.
