# Initial startup recovery across admitted cohorts

The private initial-owner fixture now covers reservation-before-issuance,
issuance-before-construction, Omabit-only, Theme-only and DMs-only startup. The
existing full-set initial and adopted/binding recovery scenarios also regress.
Only qualification fixtures changed; core and217 test sources remain identical
to the passing M309 suite. No persistent format changes or public commands.

Each of the six initial cases kills the original shell and five controllers,
retains the original guardian and admission evidence, refuses unverified restart
and wrong-owner cleanup, then adopts one replacement per supported plugin. Before
issuance, both admission files remain missing in the abandoned directory/history;
a successful recovery never fabricates their existence. After issuance, the exact
empty or partial claim cohort remains unchanged. Theme/DMs deletion refusal keeps
the prepared cleanup intent and blocks replacement until verified. The no-transport
cases prove empty-cohort cleanup without inventing failed network work.

| Original startup | Primaries constructed total | Subscriptions | Closed channels |
| --- | ---: | ---: | ---: |
| Before issuance | 3 | 5 | 4 |
| Issued, none constructed | 3 | 5 | 4 |
| Omabit only | 4 | 5 | 4 |
| Theme only | 4 | 6 | 5 |
| DMs only | 4 | 9 | 5 |
| Full supported set | 6 | 10 | 6 |

All six sessions remain unchanged without login. DMs-only/full-set cases preserve
an actually unresolved outgoing message byte-for-byte without resend. Where DMs
never started, absence of its outbound journal is checked through recovery; the
final healthy replacement then creates a new test message. That is not evidence
of preserving an earlier message. Theme-only readiness includes its actual private
appearance; replacement resumes the same stable profile. Exact cleanup/restart
retries issue no duplicate HTTP or construction effects.

The adopted-startup regression still passes seven crashes, stable/dev/stable,
15 subscriptions and ten closed channels with unchanged sessions and unknown DM.
Independent journal replay, original recovery-record object/hash, guardian closure,
claim cohort and final guard checks pass.37 controller kills across seven completed
runs,14 original guardian receipts, eight inactive scopes, and eight clean private
compositor exits are retained. The first before-issuance trial failed because a
fixture metadata loop shadowed the selected cohort; that trial and correction are
preserved. No core defect was demonstrated by this slice.

M310 supersedes M308/M309's early/partial initial startup limit for these explicit
cohorts in the same still-live physical compositor. It does not claim general
repair of damaged admission files, arbitrary startup interleavings, interrupted
authority provisioning, compositor/logout/reboot recovery or cross-session
bootstrap. Compatible immutable-generation preflight and supported public profile
workflows remain next gates. Synthetic Eyre/private keyring/headless Sway evidence
does not qualify installed Hyprland/full shell. Production ships, services and
plugins, tmux ships and temp-ur remained excluded. Exports omit keyrings,
account/application files, piers and raw credentials.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-initial-startup-cohorts
