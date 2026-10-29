# Application connections and future wallet research

Recorded 2026-10-21; research register, not a claim of implemented isolation.
Product context: [platform roadmap](platform-roadmap.md).

## Established constraints

- Omabit's current architecture explicitly assumes unsandboxed Omarchy plugins.
  Code running as the same desktop user can exercise that user's authority.
  A plugin manifest, app ID, SKILL.md or approval button is not an OS sandbox.
- Collector Secret Service storage reduces accidental secret exposure. A locked
  keyring is an error, not grounds to fall back silently to plaintext. It does
  not establish isolation from hostile code running under the same account.
- The collector's headless file adapter explicitly stores plaintext with private
  file permissions. Tend also documents private cookie files. Talon's current
  desktop SessionStore documents plaintext JSON. These are observations to review,
  not security recommendations to copy into a new connection service.
- An owner Eyre cookie has broad ship authority. The first release intentionally
  hands a different ordinary session to each plugin/profile and keeps data traffic
  direct. There is no per-desk enforcement or collector authorization proxy.
  Separate sessions provide independent logout, not isolation between desks.
- Lattice's agent-key implementation stores a salted token hash and gates selected
  routes by scope. It supports revocation. That is a useful ship-side enforcement
  pattern, not a universal capability system for all Gall agents.
- Grubbery app/nexus permissions govern backend authority. They are distinct from
  per-client tokens and local plugin consent. Existing broad permissions can allow
  new features on upgrade without a new prompt; review the release contract.
- A runner administrator can reach the pier and obtain owner authority. Multiple
  fleet operators therefore do not imply private app data isolated from the host
  administrator. Their individual management grants still need revocation/audit.

## Initial connection contract (user correction, 2026-10-21)

Theme and DMs are the first pair. Plugins own their session storage and direct
transport, and support URL + `+code` login without Omabit. Omabit helps approve
and create a fresh session; it does not serve application data. Do not persist
`+code` merely to create later sessions. Qualify the authenticated `%base`
code-thread route found in the inspected source with transient secret handling and an independent login client.

Inter-desk capabilities and the user's 407k / Gall 2026 work are deferred. Do not
make the speculative isolation questions below blockers for the initial trusted
desktop integration. Current work must still avoid accidental credential leaks,
mixing profiles/origins and revoking another plugin's session.

## Future research questions

| Question | Evidence required before claiming protection |
| --- | --- |
| Can one enabled plugin impersonate another? | Actual shell process model, peer credential handling and attempted bypass of app-ID-only policy |
| Can an unapproved same-user process read a cookie or invoke the collector? | Keyring ACL/unlock tests, socket policy and process isolation experiment |
| Does revoking a plugin session stop its access? | Its watches and later requests fail while the other plugin and original session remain valid; distinguish local clearing from confirmed server logout |
| Does an endpoint change leak sessions? | Wrong origin, redirect, hostname/port change, different identity and repeated fake `~zod` fixtures |
| Can backend updates exceed the approved access? | Manifest and protocol changes, existing broad permits, new routes and a renewed-consent test |
| How can native/PWA clients reuse login? | Explicit supported handoff per client; no cookie in URL, argv, environment or a universal shared credential file |

The first shared-connection release may intentionally support trusted/cooperative
desktop plugins. Label that trust model plainly. Stronger claims require a real
boundary (for example a constrained separate process and/or ship-scoped tokens),
plus an adversarial test, rather than additional UI text.

## Wallet watchlist — outside current implementation scope

Keep these concerns visible while building connections:

- Separate wallet seeds/private signing keys, Urbit identity/networking keys,
  SSH management keys, owner Eyre sessions and application tokens. They serve
  different purposes and must not inherit each other's permissions.
- Prefer an explicit signer interface over handing private keys to QML/plugins.
  Evaluate external/hardware signers and OS/TPM-backed storage against actual
  exportability and unattended-signing needs before selecting a mechanism.
- Investigate encrypted-at-rest storage, unlock lifetime, suspend/lock behavior,
  recovery/backup, loss, rotation and revocation. Encryption at rest does not
  prevent an unlocked compromised process from asking a signer to act.
- Bind each approval to an exact signing operation and appropriate domain/chain,
  human-readable intent, replay constraints and expiry. A general application
  connection approval must never silently become permission to move funds.
- Keep signing secrets out of logs, clipboard history, shell arguments, crash
  dumps and ordinary plugin settings. Test these channels with synthetic secrets.
- Treat ship-based message signing and asset custody as different interfaces;
  existing Talon/Lattice signing support does not by itself establish a suitable
  wallet custody design.

No wallet, key migration, signing operation or security guarantee is delivered by
this planning work. Revisit this register after each connection/security experiment.


The [real approval-flow gate](diagnostics/2026-10-21-public-auth-flow.md) demonstrates
independent session revocation and direct plugin transport after parent logout and
enrollment removal. It does not establish desk isolation or a wallet key boundary.
Profile/keyring attribute namespaces prevent accidental instance mixing; they do
not isolate malicious plugins sharing a Unix account and Secret Service access.
The lab relies on separate filesystem/process/network namespaces and a private
bus/keyring. Full native plugin/profile isolation and future wallet key protection
remain separate work; no wallet or cryptographic key store was implemented.


## Private project networks (2026-10-22)

Pinned Vere's fake network uses a shared loopback namespace. Its per-ship Lens
servers also live there, beginning at 12321 and incrementing on conflicts. A
participant can therefore reach other project ships' administrative endpoints;
separate HTTP ports and separate Eyre cookies do not create a hostile-peer
boundary. Keep this model limited to an explicitly trusted development project.
The first probe uses only newly created fake identities and a no-NIC VM; no real
wallet, identity key or existing ship is admitted. Future wallet/signing designs
must not use project membership or an application-connection approval as authority
to access signing keys. No wallet mechanism is implemented by this research.

## Collection loader lifecycle (2026-10-24)

M213 reproduces service-object overlap during a one-pass collection switch and
old code retained by `keepLoaded` despite refreshed manifest metadata. Explicit
helper teardown and loaded-generation checks prevent accidental mixing in the
synthetic test. They do not isolate malicious plugins or protect wallet keys from
same-user code. The native self-scoped plugin APIs are useful interface boundaries,
not an OS sandbox. See [loader evidence](diagnostics/2026-10-24-collection-loader.md).

## Keyed private-chain and real-helper qualification (2026-10-24)

M234's distinct collection device IDs and helper teardown prevent accidental
cross-profile activity; all code still shares the test Unix user's authority.
M235–M237's Lens shutdown explicitly uses broad loopback administrative authority
inside a trusted private project. Successful Eyre cookie isolation does not reduce
that administrative access. Fresh Urbit network keys are private test inputs,
separate from the public development chain mnemonic. Both are outside any wallet
custody design; neither application approval nor collection activation grants
permission to access a future wallet signer. These observations extend the earlier
security register without claiming a new OS or per-desk boundary.

## Native appearance process authority (2026-10-24)

M264's copied font/restart scripts operate on shell processes and user fontconfig,
not only plugin-local state. A profile label or helper subprocess does not reduce
same-user authority. The test's boundary is its private OS namespaces and inert
systemd/Hyprland adapters; this is not protection supplied by Omarchy plugin IDs.
Keep a future wallet signer and consent lifetime independent of shell/plugin
restart. No wallet custody mechanism or new isolation guarantee is introduced.
See [the lifecycle evidence](diagnostics/2026-10-24-native-appearance-effects.md).


## Credential-store lock boundary (M365, 2026-10-25)

The actual Theme/DMs store helpers refuse new reads and replacement saves while
an isolated Secret Service collection is locked. Four independent synthetic
sessions survive an owned daemon restart with the original password, without
plaintext fallback in state/config/cache. Failed command-line unlock attempts
remain distinct from the successful daemon-restart path; UI unlock and a full
login/reboot are unqualified. See [the test and limits](diagnostics/2026-10-25-plugin-keyring-lock.md).

This is storage-availability evidence, not a wallet or application permission
boundary. It does not demonstrate revocation of cookies already loaded in memory,
termination of existing Eyre channels, or prevention of already-authorized signing.
Future wallet work must test those lifetimes separately and retain per-operation
signer approval. Plugin/profile attribute names still do not isolate hostile code
running as the same Unix user. No wallet keys or signing operations were tested.


## Bootstrap-key integrity is distinct from wallet custody (M414, 2026-10-26)

The member-key source verifier holds private files and compares accepted bytes,
ownership and stable metadata. Its synthetic tests demonstrate refusal of
accidental substitution, not protection from hostile same-UID plugins, secure
memory erasure, cryptographic key/chain correspondence or wallet signing consent.
Do not repurpose an accepted networking-key hash or broad Eyre session as wallet
authorization. Keep the future signer boundary and per-operation consent separate.
No actual wallet/network keys were read by this source qualification. See
[the verifier and limits](diagnostics/2026-10-26-private-member-key.md).


## Key-proof process boundary (M419, 2026-10-26)

The private member proof source sends key-bearing Hoon through child stdin and
suppresses evaluator diagnostics, returning a bound file hash and public chain
evidence only. This reduces accidental argument/log disclosure; it does not
protect against same-UID process inspection, clear all immutable strings from
memory, or provide wallet signing authorization. Its current tests use synthetic
keys and a stub evaluator. Keep any future wallet signer and consent boundary
separate. See [source evidence and limits](diagnostics/2026-10-26-private-member-key-proof.md).
